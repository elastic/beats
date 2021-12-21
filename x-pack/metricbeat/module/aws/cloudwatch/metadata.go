// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudwatch

import (
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
func addMetadata(namespace string, endpoint string, regionName string, awsConfig awssdk.Config, fips_enabled bool, events map[string]mb.Event) map[string]mb.Event {
	switch namespace {
	case namespaceEC2:
		return ec2.AddMetadata(endpoint, regionName, awsConfig, fips_enabled, events)
	case namespaceRDS:
		return rds.AddMetadata(endpoint, regionName, awsConfig, fips_enabled, events)
	case namespaceSQS:
		return sqs.AddMetadata(endpoint, regionName, awsConfig, fips_enabled, events)
	default:
		return events
	}
}
