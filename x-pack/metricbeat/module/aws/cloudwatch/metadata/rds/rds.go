// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package rds

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/rds/types"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
)

const metadataPrefix = "aws.rds.db_instance."

// AddMetadata adds metadata for RDS instances from a specific region
func AddMetadata(regionName string, awsConfig awssdk.Config, fips_enabled bool, events map[string]mb.Event) (map[string]mb.Event, error) {
	svc := rds.NewFromConfig(awsConfig, func(o *rds.Options) {
		if fips_enabled {
			o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
		}
	})

	// Get DBInstance IDs per region
	dbDetailsMap, err := getDBInstancesPerRegion(svc)
	if err != nil {
		logp.Error(fmt.Errorf("getInstancesPerRegion failed, skipping region %s: %w", regionName, err))
		return events, nil
	}

	for _, event := range events {
		cpuValue, err := event.RootFields.GetValue("aws.rds.metrics.CPUUtilization.avg")
		if err == nil {
			if value, ok := cpuValue.(float64); ok {
				_, _ = event.RootFields.Put("aws.rds.metrics.CPUUtilization.avg", value/100)
			}
		}
	}

	for identifier, output := range dbDetailsMap {
		if _, ok := events[identifier]; !ok {
			continue
		}

		if output.DBInstanceArn != nil {
			_, _ = events[identifier].RootFields.Put(metadataPrefix+"arn", *output.DBInstanceArn)
		}

		if output.DBInstanceStatus != nil {
			_, _ = events[identifier].RootFields.Put(metadataPrefix+"status", *output.DBInstanceStatus)
		}

		if output.DBInstanceIdentifier != nil {
			_, _ = events[identifier].RootFields.Put(metadataPrefix+"identifier", *output.DBInstanceIdentifier)
		}

		if output.DBClusterIdentifier != nil {
			_, _ = events[identifier].RootFields.Put(metadataPrefix+"db_cluster_identifier", *output.DBClusterIdentifier)
		}

		if output.DBInstanceClass != nil {
			_, _ = events[identifier].RootFields.Put(metadataPrefix+"class", *output.DBInstanceClass)
		}

		if output.Engine != nil {
			_, _ = events[identifier].RootFields.Put(metadataPrefix+"engine_name", *output.Engine)
		}

		if output.AvailabilityZone != nil {
			_, _ = events[identifier].RootFields.Put("cloud.availability_zone", *output.AvailabilityZone)
		}
	}
	return events, nil
}

func getDBInstancesPerRegion(svc *rds.Client) (map[string]*types.DBInstance, error) {
	describeInstanceInput := &rds.DescribeDBInstancesInput{}

	output, err := svc.DescribeDBInstances(context.TODO(), describeInstanceInput)
	if err != nil {
		return nil, fmt.Errorf("error DescribeDBInstancesRequest: %w", err)
	}

	instancesOutputs := map[string]*types.DBInstance{}
	for _, dbInstance := range output.DBInstances {
		instance := dbInstance
		instancesOutputs[*instance.DBInstanceIdentifier] = &instance
	}
	return instancesOutputs, nil
}
