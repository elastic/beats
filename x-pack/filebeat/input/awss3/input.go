// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/elastic/beats/v7/filebeat/beater"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/go-concert/unison"
)

const (
	inputName                = "aws-s3"
	sqsAccessDeniedErrorCode = "AccessDeniedException"
)

func Plugin(store beater.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "Collect logs from s3",
		Manager:    &s3InputManager{store: store},
	}
}

type s3InputManager struct {
	store beater.StateStore
}

// s3Input is a input for reading logs from S3 when triggered by an SQS message.
type s3Input struct {
	config    config
	awsConfig awssdk.Config
	store     beater.StateStore
	metrics   *inputMetrics
}

func (im *s3InputManager) Init(grp unison.Group, mode v2.Mode) error {
	return nil
}

func (im *s3InputManager) Create(cfg *conf.C) (v2.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return newInput(config, im.store)
}

func newInput(config config, store beater.StateStore) (v2.Input, error) {
	awsConfig, err := awscommon.InitializeAWSConfig(config.AWSConfig)

	if config.AWSConfig.Endpoint != "" {
		// Add a custom endpointResolver to the awsConfig so that all the requests are routed to this endpoint
		awsConfig.EndpointResolverWithOptions = awssdk.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (awssdk.Endpoint, error) {
			return awssdk.Endpoint{
				PartitionID:   "aws",
				URL:           config.AWSConfig.Endpoint,
				SigningRegion: awsConfig.Region,
			}, nil
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS credentials: %w", err)
	}

	if config.QueueURL != "" {
		return newSQSReaderInput(config, awsConfig, store)
		//return in.runQueueReader(ctx, inputContext, pipeline)
	}

	if config.BucketARN != "" || config.NonAWSBucketName != "" {
		return newS3PollerInput(config, awsConfig, store)
		//return in.runS3Poller(ctx, inputContext, pipeline)
	}

	return nil, fmt.Errorf("configuration has no SQS queue URL and no S3 bucket ARN")

	// return &s3Input{
	// 	config:    config,
	// 	awsConfig: awsConfig,
	// 	store:     store,
	// }, nil
}

// boolPtr returns a pointer to b.
func boolPtr(b bool) *bool { return &b }
