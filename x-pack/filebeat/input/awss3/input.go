// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"
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

	if config.AccessPointARN != "" {
		// When using the access point ARN, requests must be directed to the
		// access point hostname. The access point hostname takes the form
		// AccessPointName-AccountId.s3-accesspoint.Region.amazonaws.com
		arnParts := strings.Split(config.AccessPointARN, ":")
		region := arnParts[3]
		accountID := arnParts[4]
		accessPointName := strings.Split(arnParts[5], "/")[1]

		// Construct the endpoint for the Access Point
		endpoint := fmt.Sprintf("%s-%s.s3-accesspoint.%s.amazonaws.com", accessPointName, accountID, region)

		// Set up a custom endpoint resolver for Access Points
		//nolint:staticcheck // haven't migrated to the new interface yet
		awsConfig.EndpointResolverWithOptions = awssdk.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (awssdk.Endpoint, error) {
			return awssdk.Endpoint{
				URL:               fmt.Sprintf("https://%s", endpoint),
				SigningRegion:     region,
				HostnameImmutable: true,
			}, nil
		})
		awsConfig.Region = region
	}

	if config.AccessPointARN == "" && config.RegionName != "" {
		// The awsConfig now contains the region from the credential profile or default region
		// if the region is explicitly set in the config, then it wins
		awsConfig.Region = config.RegionName
	}

	if config.QueueURL != "" {
		return newSQSReaderInput(config, awsConfig), nil
	}

	if config.BucketARN != "" || config.AccessPointARN != "" || config.NonAWSBucketName != "" {
		return newS3PollerInput(config, awsConfig, im.store)
	}

	return nil, fmt.Errorf("configuration has no SQS queue URL and no S3 bucket ARN")
}

// boolPtr returns a pointer to b.
func boolPtr(b bool) *bool { return &b }
