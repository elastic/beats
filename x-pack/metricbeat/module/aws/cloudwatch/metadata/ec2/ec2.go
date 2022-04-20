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
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

const metadataPrefix = "aws.ec2.instance."

// AddMetadata adds metadata for EC2 instances from a specific region
func AddMetadata(endpoint string, regionName string, awsConfig awssdk.Config, fips_enabled bool, events map[string]mb.Event) (map[string]mb.Event, error) {
	ec2ServiceName := awscommon.CreateServiceName("ec2", fips_enabled, regionName)
	svcEC2 := ec2.NewFromConfig(awscommon.EnrichAWSConfigWithEndpoint(
		endpoint, ec2ServiceName, regionName, awsConfig))

	instancesOutputs, err := getInstancesPerRegion(svcEC2)
	if err != nil {
		return events, fmt.Errorf("getInstancesPerRegion failed, skipping region %s: %w", regionName, err)
	}

	// collect monitoring state for each instance
	monitoringStates := map[string]string{}
	for instanceID, output := range instancesOutputs {
		if _, ok := events[instanceID]; !ok {
			continue
		}

		for _, tag := range output.Tags {
			if *tag.Key == "Name" {
				events[instanceID].RootFields.Put("cloud.instance.name", *tag.Value)
				events[instanceID].RootFields.Put("host.name", *tag.Value)
			}
		}

		events[instanceID].RootFields.Put("cloud.instance.id", instanceID)

		if output.InstanceType != "" {
			events[instanceID].RootFields.Put("cloud.machine.type", output.InstanceType)
		} else {
			logp.Error(fmt.Errorf("InstanceType is empty"))
		}

		placement := output.Placement
		if placement != nil {
			events[instanceID].RootFields.Put("cloud.availability_zone", *placement.AvailabilityZone)
		}

		if output.State.Name != "" {
			events[instanceID].RootFields.Put(metadataPrefix+"state.name", output.State.Name)
		} else {
			logp.Error(fmt.Errorf("instance.State.Name is empty"))
		}

		if output.Monitoring.State != ""{
			monitoringStates[instanceID] = string(output.Monitoring.State)
			events[instanceID].RootFields.Put(metadataPrefix+"monitoring.state", output.Monitoring.State)
		} else {
			logp.Error(fmt.Errorf("Monitoring.State is empty"))
		}

		cpuOptions := output.CpuOptions
		if cpuOptions != nil {
			events[instanceID].RootFields.Put(metadataPrefix+"core.count", *cpuOptions.CoreCount)
			events[instanceID].RootFields.Put(metadataPrefix+"threads_per_core", *cpuOptions.ThreadsPerCore)
		}

		publicIP := output.PublicIpAddress
		if publicIP != nil {
			events[instanceID].RootFields.Put(metadataPrefix+"public.ip", *publicIP)
		}

		privateIP := output.PrivateIpAddress
		if privateIP != nil {
			events[instanceID].RootFields.Put(metadataPrefix+"private.ip", *privateIP)
		}

		events[instanceID].RootFields.Put(metadataPrefix+"image.id", *output.ImageId)
		events[instanceID].RootFields.Put(metadataPrefix+"state.code", *output.State.Code)
		events[instanceID].RootFields.Put(metadataPrefix+"public.dns_name", *output.PublicDnsName)
		events[instanceID].RootFields.Put(metadataPrefix+"private.dns_name", *output.PrivateDnsName)

		// add host cpu/network/disk fields and host.id
		addHostFields(events[instanceID], instanceID)

		// add rate metrics
		calculateRate(events[instanceID], monitoringStates[instanceID])
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
			err = errors.Wrap(err, "Error DescribeInstances")
			return nil, err
		}

		for _, reservation := range output.Reservations {
			for _, instance := range reservation.Instances {
				instancesOutputs[*instance.InstanceId] = &instance
			}
		}
	}
	return instancesOutputs, nil
}

func addHostFields(event mb.Event, instanceID string) {
	event.RootFields.Put("host.id", instanceID)

	// If there is no instance name, use instance ID as the host.name
	hostName, err := event.RootFields.GetValue("host.name")
	if err == nil && hostName != nil {
		event.RootFields.Put("host.name", hostName)
	} else {
		event.RootFields.Put("host.name", instanceID)
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
			if ec2MetricName == "cpu.total.pct" {
				value = value / 100
			}
			event.RootFields.Put(hostMetricName, value)
		}
	}
}

func calculateRate(event mb.Event, monitoringState string) {
	var period = 300.0
	if monitoringState != "disabled" {
		period = 60.0
	}

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
			rateValue := metricValue.(float64) / period
			event.RootFields.Put(strings.Replace(metricName, ".sum", ".rate", -1), rateValue)
		}
	}
}
