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
	NamespaceEC2 = "AWS/EC2"
)

// NewMetadata returns a service to fetch metadata from a config struct.
func NewMetadata(namespace string, endpoint string, regionName string, awsConfig awssdk.Config, events map[string]mb.Event) map[string]mb.Event {
	switch namespace {
	case NamespaceEC2:
		return ec2.NewMetadataService(endpoint, regionName, awsConfig, events)
	default:
		return events
	}
}
