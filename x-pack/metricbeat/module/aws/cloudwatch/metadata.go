// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudwatch

import (
	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/cloudwatch/ec2"
)

// AWS namespaces
const (
	namespaceEC2 = "AWS/EC2"
)

// addMetadata returns a service to fetch metadata from a config struct.
func addMetadata(namespace string, endpoint string, regionName string, awsConfig awssdk.Config, events map[string]mb.Event) map[string]mb.Event {
	switch namespace {
	case namespaceEC2:
		return ec2.AddMetadata(endpoint, regionName, awsConfig, events)
	default:
		return events
	}
}
