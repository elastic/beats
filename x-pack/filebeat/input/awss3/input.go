// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/features"
	"github.com/elastic/beats/v7/libbeat/statestore"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
	"github.com/elastic/go-concert/unison"
)

const inputName = "aws-s3"

func Plugin(logger *logp.Logger, store statestore.States, p *paths.Path) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "Collect logs from s3",
		Manager:    &s3InputManager{store: store, logger: logger, path: p},
	}
}

type s3InputManager struct {
	store  statestore.States
	logger *logp.Logger
	path   *paths.Path
}

func (im *s3InputManager) Init(grp unison.Group) error {
	return nil
}

func (im *s3InputManager) Create(cfg *conf.C) (v2.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	if features.AwsS3V2() {
		return newInputV2(config, im.store, im.path, im.logger)
	}

	// Legacy path — deprecated. When removing this code path, delete these
	// legacy-only files (and their corresponding _test.go files):
	//
	//   sqs_input.go        — sqsReaderInput, sqsWorker orchestration
	//   s3_input.go         — s3PollerInput orchestration
	//   s3_objects.go       — s3ObjectProcessor, s3ObjectProcessorFactory
	//   polling_strategy.go — normalPollingStrategy, lexicographicalPollingStrategy
	//
	// Also delete input_integration_test.go and input_benchmark_test.go, and
	// remove the legacy branch below (+ this comment) from this file.
	//
	// Shared files reused by V2 (keep): interfaces.go, s3.go, sqs.go,
	// sqs_s3_event.go, acks.go, metrics.go, state.go, states.go, config.go,
	// script.go, script_session.go, script_jss3event_v2.go, s3_filters.go.
	im.logger.Warn("The legacy aws-s3 input implementation is deprecated and will be removed in a future release. " +
		"Remove 'features.aws_s3_v2.enabled: false' from your configuration to use the new implementation.")

	awsConfig, err := awscommon.InitializeAWSConfig(config.AWSConfig, im.logger)
	if err != nil {
		return nil, fmt.Errorf("initializing AWS config: %w", err)
	}

	if config.RegionName != "" {
		// The awsConfig now contains the region from the credential profile or default region
		// if the region is explicitly set in the config, then it wins
		awsConfig.Region = config.RegionName
	}

	if config.QueueURL != "" {
		return newSQSReaderInput(config, awsConfig, im.path), nil
	}

	if config.BucketARN != "" || config.AccessPointARN != "" || config.NonAWSBucketName != "" {
		return newS3PollerInput(config, awsConfig, im.store)
	}

	return nil, fmt.Errorf("configuration has no SQS queue URL and no S3 bucket ARN")
}

// boolPtr returns a pointer to b.
func boolPtr(b bool) *bool { return &b }
