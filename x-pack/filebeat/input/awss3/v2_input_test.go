// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/features"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
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
	cfg := conf.MustNewConfigFrom(map[string]interface{}{
		"features": map[string]interface{}{
			"aws_s3_v2": map[string]interface{}{"enabled": true},
		},
	})
	require.NoError(t, features.UpdateFromConfig(cfg))
	t.Cleanup(func() {
		off := conf.MustNewConfigFrom(map[string]interface{}{
			"features": map[string]interface{}{
				"aws_s3_v2": map[string]interface{}{"enabled": false},
			},
		})
		_ = features.UpdateFromConfig(off)
	})

	assert.True(t, features.AwsS3V2(), "V2 flag should be enabled")

	inputCfg := conf.MustNewConfigFrom(map[string]interface{}{
		"queue_url": "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue",
	})
	in, err := Plugin(logp.NewLogger(inputName), openTestStatestore(), nil).Manager.Create(inputCfg)
	require.NoError(t, err, "Plugin.Create should succeed")
	_, ok := in.(*inputV2)
	assert.True(t, ok, "expected *inputV2 when flag is enabled, got %T", in)
}
