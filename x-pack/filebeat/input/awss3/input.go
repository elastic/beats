// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"
	"net/url"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/elastic/beats/v7/filebeat/beater"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/go-concert/unison"
)

const inputName = "aws-s3"

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

func (im *s3InputManager) Init(grp unison.Group) error {
	return nil
}

func (im *s3InputManager) Create(cfg *conf.C) (v2.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	awsConfig, err := awscommon.InitializeAWSConfig(config.AWSConfig)
	if err != nil {
		return nil, fmt.Errorf("initializing AWS config: %w", err)
	}

	endpointUri, err := url.Parse(config.AWSConfig.Endpoint)
	// A custom endpoint has been specified!
	if err == nil && config.AWSConfig.Endpoint != "" && !strings.HasPrefix(endpointUri.Hostname(), "s3") {

		// For backwards compat:
		// If the endpoint does not start with S3, we will use the endpoint resolver to make all SDK requests use the specified endpoint
		// If the endpoint does start with S3, we will use the default resolver uses the endpoint field but can replace s3 with the desired service name like sqs

		awsConfig.EndpointResolverWithOptions = awssdk.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (awssdk.Endpoint, error) {
			return awssdk.Endpoint{
				PartitionID:       "aws",
				Source:            awssdk.EndpointSourceCustom,
				URL:               config.AWSConfig.Endpoint,
				SigningRegion:     awsConfig.Region,
				HostnameImmutable: true,
			}, nil
		})
	}

	if config.QueueURL != "" {
		return newSQSReaderInput(config, awsConfig), nil
	}

	if config.BucketARN != "" || config.NonAWSBucketName != "" {
		return newS3PollerInput(config, awsConfig, im.store)
	}

	return nil, fmt.Errorf("configuration has no SQS queue URL and no S3 bucket ARN")
}

// boolPtr returns a pointer to b.
func boolPtr(b bool) *bool { return &b }
