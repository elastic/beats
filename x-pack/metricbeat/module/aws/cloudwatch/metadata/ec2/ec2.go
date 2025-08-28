// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"context"
	"fmt"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
	"github.com/elastic/elastic-agent-libs/logp"
)

const metadataPrefix = "aws.ec2.instance."

// AddMetadata adds metadata for EC2 instances from a specific region
func AddMetadata(logger *logp.Logger, regionName string, awsConfig awssdk.Config, fips_enabled bool, events map[string]mb.Event) (map[string]mb.Event, error) {
	svcEC2 := ec2.NewFromConfig(awsConfig, func(o *ec2.Options) {
		if fips_enabled {
			o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
		}

	})

	instancesOutputs, err := getInstancesPerRegion(svcEC2)
	if err != nil {
		return events, fmt.Errorf("aws.ec2.instance fields are not available, skipping region %s: %w", regionName, err)
	}

	for eventIdentifier := range events {
		eventIdentifierComponents := strings.Split(eventIdentifier, "-")
		potentialInstanceID := strings.Join(eventIdentifierComponents[0:len(eventIdentifierComponents)-1], "-")

		// add host cpu/network/disk fields and host.id and rate metrics for all instances from both the monitoring
		// account and linked source accounts if include_linked_accounts is set to true
		addHostFields(events[eventIdentifier], potentialInstanceID)
		period, err := events[eventIdentifier].RootFields.GetValue(aws.CloudWatchPeriodName)
		if err != nil {
			logger.Warnf("can't get period information for instance %s, skipping rate calculation", eventIdentifier)
		} else {
			calculateRate(events[eventIdentifier], period.(int))
		}

		// add instance ID from dimension value
		if dimInstanceID, err := events[eventIdentifier].RootFields.GetValue("aws.dimensions.InstanceId"); err == nil {
			_, _ = events[eventIdentifier].RootFields.Put("cloud.instance.id", dimInstanceID)
		}

		for instanceID, output := range instancesOutputs {
			if instanceID != potentialInstanceID {
				continue
			}
			for _, tag := range output.Tags {
				if *tag.Key == "Name" {
					_, _ = events[eventIdentifier].RootFields.Put("cloud.instance.name", *tag.Value)
					_, _ = events[eventIdentifier].RootFields.Put("host.name", *tag.Value)
				}
			}

			if output.InstanceType != "" {
				_, _ = events[eventIdentifier].RootFields.Put("cloud.machine.type", output.InstanceType)
			} else {
				logger.Error("InstanceType is empty")
			}

			placement := output.Placement
			if placement != nil {
				_, _ = events[eventIdentifier].RootFields.Put("cloud.availability_zone", *placement.AvailabilityZone)
			}

			if output.State.Name != "" {
				_, _ = events[eventIdentifier].RootFields.Put(metadataPrefix+"state.name", output.State.Name)
			} else {
				logger.Error("instance.State.Name is empty")
			}

			if output.Monitoring.State != "" {
				_, _ = events[eventIdentifier].RootFields.Put(metadataPrefix+"monitoring.state", output.Monitoring.State)
			} else {
				logger.Error("Monitoring.State is empty")
			}

			cpuOptions := output.CpuOptions
			if cpuOptions != nil {
				_, _ = events[eventIdentifier].RootFields.Put(metadataPrefix+"core.count", *cpuOptions.CoreCount)
				_, _ = events[eventIdentifier].RootFields.Put(metadataPrefix+"threads_per_core", *cpuOptions.ThreadsPerCore)
			}

			publicIP := output.PublicIpAddress
			if publicIP != nil {
				_, _ = events[eventIdentifier].RootFields.Put(metadataPrefix+"public.ip", *publicIP)
			}

			privateIP := output.PrivateIpAddress
			if privateIP != nil {
				_, _ = events[eventIdentifier].RootFields.Put(metadataPrefix+"private.ip", *privateIP)
			}

			_, _ = events[eventIdentifier].RootFields.Put(metadataPrefix+"image.id", *output.ImageId)
			_, _ = events[eventIdentifier].RootFields.Put(metadataPrefix+"state.code", *output.State.Code)
			_, _ = events[eventIdentifier].RootFields.Put(metadataPrefix+"public.dns_name", *output.PublicDnsName)
			_, _ = events[eventIdentifier].RootFields.Put(metadataPrefix+"private.dns_name", *output.PrivateDnsName)
		}
	}

	return events, nil
}

func getInstancesPerRegion(svc *ec2.Client) (map[string]*ec2types.Instance, error) {
	instancesOutputs := map[string]*ec2types.Instance{}
	output := ec2.DescribeInstancesOutput{NextToken: nil}
	init := true
	for init || output.NextToken != nil {
		init = false
		describeInstanceInput := &ec2.DescribeInstancesInput{}
		output, err := svc.DescribeInstances(context.Background(), describeInstanceInput)
		if err != nil {
			err = fmt.Errorf("error DescribeInstances: %w", err)
			return nil, err
		}

		for _, reservation := range output.Reservations {
			for i := range reservation.Instances {
				instance := reservation.Instances[i]
				instancesOutputs[*instance.InstanceId] = &instance
			}
		}
	}
	return instancesOutputs, nil
}

func addHostFields(event mb.Event, instanceID string) {
	_, _ = event.RootFields.Put("host.id", instanceID)

	// If there is no instance name, use instance ID as the host.name
	hostName, err := event.RootFields.GetValue("host.name")
	if err == nil && hostName != nil {
		_, _ = event.RootFields.Put("host.name", hostName)
	} else {
		_, _ = event.RootFields.Put("host.name", instanceID)
	}

	hostFieldTable := map[string]string{
		"aws.ec2.metrics.CPUUtilization.avg":    "host.cpu.usage",
		"aws.ec2.metrics.NetworkIn.sum":         "host.network.ingress.bytes",
		"aws.ec2.metrics.NetworkOut.sum":        "host.network.egress.bytes",
		"aws.ec2.metrics.NetworkPacketsIn.sum":  "host.network.ingress.packets",
		"aws.ec2.metrics.NetworkPacketsOut.sum": "host.network.egress.packets",
		"aws.ec2.metrics.DiskReadBytes.sum":     "host.disk.read.bytes",
		"aws.ec2.metrics.DiskWriteBytes.sum":    "host.disk.write.bytes",
	}

	for ec2MetricName, hostMetricName := range hostFieldTable {
		metricValue, err := event.RootFields.GetValue(ec2MetricName)
		if err != nil {
			continue
		}

		if value, ok := metricValue.(float64); ok {
			if ec2MetricName == "aws.ec2.metrics.CPUUtilization.avg" {
				value = value / 100
			}
			_, _ = event.RootFields.Put(hostMetricName, value)
		}
	}
}

func calculateRate(event mb.Event, periodInSeconds int) {
	metricList := []string{
		"aws.ec2.metrics.NetworkIn.sum",
		"aws.ec2.metrics.NetworkOut.sum",
		"aws.ec2.metrics.NetworkPacketsIn.sum",
		"aws.ec2.metrics.NetworkPacketsOut.sum",
		"aws.ec2.metrics.DiskReadBytes.sum",
		"aws.ec2.metrics.DiskWriteBytes.sum",
		"aws.ec2.metrics.DiskReadOps.sum",
		"aws.ec2.metrics.DiskWriteOps.sum"}

	for _, metricName := range metricList {
		metricValue, err := event.RootFields.GetValue(metricName)
		if err == nil && metricValue != nil {
			rateValue := metricValue.(float64) / float64(periodInSeconds)
			_, _ = event.RootFields.Put(strings.Replace(metricName, ".sum", ".rate", -1), rateValue)
		}
	}
}
