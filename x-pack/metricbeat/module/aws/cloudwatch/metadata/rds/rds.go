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

	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	awscommon "github.com/menderesk/beats/v7/x-pack/libbeat/common/aws"
)

const metadataPrefix = "aws.rds.db_instance."

// AddMetadata adds metadata for RDS instances from a specific region
func AddMetadata(endpoint string, regionName string, awsConfig awssdk.Config, fips_enabled bool, events map[string]mb.Event) map[string]mb.Event {
	rdsServiceName := awscommon.CreateServiceName("rds", fips_enabled, regionName)
	svc := rds.New(awscommon.EnrichAWSConfigWithEndpoint(
		endpoint, rdsServiceName, regionName, awsConfig))

	// Get DBInstance IDs per region
	dbDetailsMap, err := getDBInstancesPerRegion(svc)
	if err != nil {
		logp.Error(fmt.Errorf("getInstancesPerRegion failed, skipping region %s: %w", regionName, err))
		return events
	}

	for _, event := range events {
		cpuValue, err := event.RootFields.GetValue("aws.rds.metrics.CPUUtilization.avg")
		if err == nil {
			if value, ok := cpuValue.(float64); ok {
				event.RootFields.Put("aws.rds.metrics.CPUUtilization.avg", value/100)
			}
		}
	}

	for identifier, output := range dbDetailsMap {
		if _, ok := events[identifier]; !ok {
			continue
		}

		if output.DBInstanceArn != nil {
			events[identifier].RootFields.Put(metadataPrefix+"arn", *output.DBInstanceArn)
		}

		if output.DBInstanceStatus != nil {
			events[identifier].RootFields.Put(metadataPrefix+"status", *output.DBInstanceStatus)
		}

		if output.DBInstanceIdentifier != nil {
			events[identifier].RootFields.Put(metadataPrefix+"identifier", *output.DBInstanceIdentifier)
		}

		if output.DBClusterIdentifier != nil {
			events[identifier].RootFields.Put(metadataPrefix+"db_cluster_identifier", *output.DBClusterIdentifier)
		}

		if output.DBInstanceClass != nil {
			events[identifier].RootFields.Put(metadataPrefix+"class", *output.DBInstanceClass)
		}

		if output.Engine != nil {
			events[identifier].RootFields.Put(metadataPrefix+"engine_name", *output.Engine)
		}

		if output.AvailabilityZone != nil {
			events[identifier].RootFields.Put("cloud.availability_zone", *output.AvailabilityZone)
		}
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
		instance := dbInstance
		instancesOutputs[*instance.DBInstanceIdentifier] = &instance
	}
	return instancesOutputs, nil
}
