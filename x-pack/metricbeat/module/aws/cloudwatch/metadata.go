// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudwatch

import (
	"fmt"
	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/cloudwatch/metadata/ec2"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/cloudwatch/metadata/rds"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/cloudwatch/metadata/sqs"
)

// AWS namespaces
const (
	namespaceEC2 = "AWS/EC2"
	namespaceRDS = "AWS/RDS"
	namespaceSQS = "AWS/SQS"
)

// addMetadata adds metadata to the given events map based on namespace
func addMetadata(namespace string, endpoint string, regionName string, awsConfig awssdk.Config, fipsEnabled bool, events map[string]mb.Event) (map[string]mb.Event, error) {
	switch namespace {
	case namespaceEC2:
		events, err := ec2.AddMetadata(endpoint, regionName, awsConfig, fipsEnabled, events)
		if err != nil {
			return events, fmt.Errorf("error adding metadata to ec2: %w", err)
		}
	case namespaceRDS:
		events, err := rds.AddMetadata(endpoint, regionName, awsConfig, fipsEnabled, events)
		if err != nil {
			return events, fmt.Errorf("error adding metadata to rds: %w", err)
		}
	case namespaceSQS:
		events, err := sqs.AddMetadata(endpoint, regionName, awsConfig, fipsEnabled, events)
		if err != nil {
			return events, fmt.Errorf("error adding metadata to sqs: %w", err)
		}
	default:
		return events, nil
	}

	return nil, fmt.Errorf("no events to add metadata to")
}
