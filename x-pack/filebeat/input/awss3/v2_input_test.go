// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"path/filepath"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/features"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestNewInputV2_validates_config(t *testing.T) {
	log := logp.NewNopLogger()

	t.Run("empty_config_rejected", func(t *testing.T) {
		cfg := defaultConfig()
		// No queue_url, no bucket_arn — Validate() should reject this.
		_, err := newInputV2(cfg, nil, nil, log)
		require.Error(t, err, "empty config should fail validation")
		assert.Contains(t, err.Error(), "invalid aws-s3 v2 config", "error should mention v2 config")
	})

	t.Run("sqs_mode", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.QueueURL = "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue"
		in, err := newInputV2(cfg, nil, nil, log)
		require.NoError(t, err, "valid SQS config should create input")
		assert.Equal(t, inputName, in.Name())
		assert.Equal(t, "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue", in.config.QueueURL)
	})

	t.Run("polling_mode", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.BucketARN = "arn:aws:s3:::my-test-bucket"
		in, err := newInputV2(cfg, nil, nil, log)
		require.NoError(t, err, "valid polling config should create input")
		assert.Equal(t, inputName, in.Name())
		assert.Equal(t, "arn:aws:s3:::my-test-bucket", in.config.BucketARN)
	})
}

func TestNewInputV2_non_aws_bucket_region(t *testing.T) {
	tests := []struct {
		name    string
		cfg     map[string]any
		wantErr string
	}{
		{
			name: "default_region_only",
			cfg: map[string]any{
				"non_aws_bucket_name": "r2-bucket",
				"default_region":      "auto",
			},
		},
		{
			name: "region_only",
			cfg: map[string]any{
				"non_aws_bucket_name": "r2-bucket",
				"region":              "auto",
			},
		},
		{
			name: "no_region",
			cfg: map[string]any{
				"non_aws_bucket_name": "r2-bucket",
			},
			wantErr: "region or default_region must be configured when using non_aws_bucket_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultConfig()
			err := conf.MustNewConfigFrom(tt.cfg).Unpack(&cfg)
			if err == nil {
				_, err = newInputV2(cfg, nil, nil, logp.NewNopLogger())
			}
			if tt.wantErr != "" {
				require.Error(t, err, "config should be rejected")
				assert.Contains(t, err.Error(), tt.wantErr, "error should name the missing region")
				return
			}
			assert.NoError(t, err, "config should be accepted")
		})
	}
}

// TestInputV2_non_aws_bucket_default_region checks the region that
// runPolling starts from for a non-AWS bucket that only sets default_region:
// the AWS config comes from InitializeAWSConfig and is overridden only when
// region is set, so default_region must survive.
func TestInputV2_non_aws_bucket_default_region(t *testing.T) {
	// Keep the host's AWS profile and environment from supplying a region.
	dir := t.TempDir()
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))

	cfg := defaultConfig()
	require.NoError(t, conf.MustNewConfigFrom(map[string]any{
		"non_aws_bucket_name": "r2-bucket",
		"default_region":      "auto",
	}).Unpack(&cfg), "unpacking config")

	awsCfg, err := awscommon.InitializeAWSConfig(cfg.AWSConfig, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err, "initializing AWS config")
	assert.Equal(t, "auto", awsCfg.Region, "region should come from default_region")
	assert.Empty(t, cfg.RegionName, "region should be unset")
}

func TestInputV2_resolveSQSRegion(t *testing.T) {
	tests := []struct {
		name     string
		region   string
		queueURL string
		defReg   string
		want     string
	}{
		{
			name:   "explicit_region_wins",
			region: "eu-west-1",
			want:   "eu-west-1",
		},
		{
			name:     "region_from_queue_url",
			queueURL: "https://sqs.ap-southeast-2.amazonaws.com/123456789012/q",
			want:     "ap-southeast-2",
		},
		{
			name:   "default_region_fallback",
			defReg: "us-west-2",
			want:   "us-west-2",
		},
		{
			name: "no_region_returns_empty",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := &inputV2{config: config{
				RegionName: tt.region,
				QueueURL:   tt.queueURL,
			}}
			in.config.AWSConfig.DefaultRegion = tt.defReg
			got := in.resolveSQSRegion(awssdk.Config{})
			assert.Equal(t, tt.want, got, "resolved region should match")
		})
	}
}

func TestFeatureFlag_routes_to_V2(t *testing.T) {
	cfg := conf.MustNewConfigFrom(map[string]any{
		"features": map[string]any{
			"aws_s3_v2": map[string]any{"enabled": true},
		},
	})
	require.NoError(t, features.UpdateFromConfig(cfg))
	t.Cleanup(func() {
		off := conf.MustNewConfigFrom(map[string]any{
			"features": map[string]any{
				"aws_s3_v2": map[string]any{"enabled": false},
			},
		})
		_ = features.UpdateFromConfig(off)
	})

	assert.True(t, features.AwsS3V2(), "V2 flag should be enabled")

	inputCfg := conf.MustNewConfigFrom(map[string]any{
		"queue_url": "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue",
	})
	in, err := Plugin(logptest.NewTestingLogger(t, inputName), openTestStatestore(), nil).Manager.Create(inputCfg)
	require.NoError(t, err, "Plugin.Create should succeed")
	_, ok := in.(*inputV2)
	assert.True(t, ok, "expected *inputV2 when flag is enabled, got %T", in)
}
