// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package rds

import (
	"context"
	"fmt"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/rdsiface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
)

const metadataPrefix = "aws.rds.db_instance."

// DBDetails holds detailed information from DescribeDBInstances for each rds.
type DBDetails struct {
	dbArn              string
	dbClass            string
	dbAvailabilityZone string
	dbIdentifier       string
	dbStatus           string
	tags               []aws.Tag
}

// AddMetadata adds metadata for RDS instances from a specific region
func AddMetadata(endpoint string, regionName string, awsConfig awssdk.Config, events map[string]mb.Event) map[string]mb.Event {
	svc := rds.New(awscommon.EnrichAWSConfigWithEndpoint(
		endpoint, "rds", regionName, awsConfig))

	// Get DBInstance IDs per region
	dbDetailsMap, err := getDBInstancesPerRegion(svc)
	if err != nil {
		logp.Error(fmt.Errorf("getInstancesPerRegion failed, skipping region %s: %w", regionName, err))
		return events
	}

	for identifier, output := range dbDetailsMap {
		if _, ok := events[identifier]; !ok {
			continue
		}
		events[identifier].RootFields.Put(metadataPrefix+"arn", &output.DBInstanceArn)
		events[identifier].RootFields.Put(metadataPrefix+"status", &output.DBInstanceStatus)
		events[identifier].RootFields.Put(metadataPrefix+"identifier", &output.DBInstanceIdentifier)
		events[identifier].RootFields.Put(metadataPrefix+"db_cluster_identifier", &output.DBClusterIdentifier)
		events[identifier].RootFields.Put(metadataPrefix+"class", &output.DBInstanceClass)
		events[identifier].RootFields.Put(metadataPrefix+"engine_name", &output.Engine)
		events[identifier].RootFields.Put("cloud.availability_zone", &output.AvailabilityZone)
	}
	return events
}

func getDBInstancesPerRegion(svc rdsiface.ClientAPI) (map[string]*rds.DBInstance, error) {
	describeInstanceInput := &rds.DescribeDBInstancesInput{}
	req := svc.DescribeDBInstancesRequest(describeInstanceInput)
	output, err := req.Send(context.TODO())
	if err != nil {
		return nil, errors.Wrap(err, "Error DescribeDBInstancesRequest")
	}

	instancesOutputs := map[string]*rds.DBInstance{}
	for _, dbInstance := range output.DBInstances {
		instancesOutputs[*dbInstance.DBInstanceIdentifier] = &dbInstance
	}
	return instancesOutputs, nil
}
