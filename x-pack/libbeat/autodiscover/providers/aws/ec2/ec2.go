// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	awsauto "github.com/elastic/beats/x-pack/libbeat/autodiscover/providers/aws"
)

type ec2Instance struct {
	ec2Instance ec2.Instance
	logger      *logp.Logger
}

// toMap converts this ec2Instance into the form consumed as metadata in the autodiscovery process.
func (i *ec2Instance) toMap() common.MapStr {
	instanceType, err := i.ec2Instance.InstanceType.MarshalValue()
	if err != nil {
		i.logger.Error("MarshalValue failed for instance type: ", err)
	}

	monitoringState, err := i.ec2Instance.Monitoring.State.MarshalValue()
	if err != nil {
		i.logger.Error("MarshalValue failed for monitoring state: ", err)
	}

	architecture, err := i.ec2Instance.Architecture.MarshalValue()
	if err != nil {
		i.logger.Error("MarshalValue failed for architecture: ", err)
	}

	m := common.MapStr{
		"image_id":         awsauto.SafeStrp(i.ec2Instance.ImageId),
		"vpc_id":           awsauto.SafeStrp(i.ec2Instance.VpcId),
		"subnet_id":        awsauto.SafeStrp(i.ec2Instance.SubnetId),
		"host_id":          awsauto.SafeStrp(i.ec2Instance.Placement.HostId),
		"group_name":       awsauto.SafeStrp(i.ec2Instance.Placement.GroupName),
		"arn":              awsauto.SafeStrp(i.ec2Instance.IamInstanceProfile.Arn),
		"instance_id":      awsauto.SafeStrp(i.ec2Instance.IamInstanceProfile.Id),
		"type":             instanceType,
		"private_ip":       awsauto.SafeStrp(i.ec2Instance.PrivateIpAddress),
		"private_dns_name": awsauto.SafeStrp(i.ec2Instance.PrivateDnsName),
		"public_ip":        awsauto.SafeStrp(i.ec2Instance.PublicIpAddress),
		"public_dns_name":  awsauto.SafeStrp(i.ec2Instance.PublicDnsName),
		"monitoring_state": monitoringState,
		"architecture":     architecture,
		"root_device_name": awsauto.SafeStrp(i.ec2Instance.RootDeviceName),
		"kernel_id":        awsauto.SafeStrp(i.ec2Instance.KernelId),
		"state":            i.stateMap(),
		"state_reason":     i.stateReasonMap(),
		"tags":             i.tagMap(),
	}
	return m
}

func (i *ec2Instance) toCloudMap() common.MapStr {
	m := common.MapStr{}
	availabilityZone := awsauto.SafeStrp(i.ec2Instance.Placement.AvailabilityZone)
	m["availability_zone"] = availabilityZone
	m["provider"] = "aws"

	// The region is just an AZ with the last character removed
	m["region"] = availabilityZone[:len(availabilityZone)-2]
	return m
}

// arn returns a globally unique ID. In the case of an ec2Instance, that would be its listenerArn.
func (i *ec2Instance) arn() string {
	return awsauto.SafeStrp(i.ec2Instance.IamInstanceProfile.Arn)
}

// stateMap converts the State part of the ec2 struct into a friendlier map with 'reason' and 'code' fields.
func (i *ec2Instance) stateMap() (stateMap common.MapStr) {
	state := i.ec2Instance.State
	stateMap = common.MapStr{}
	nameString, err := state.Name.MarshalValue()
	if err != nil {
		i.logger.Error("MarshalValue failed for instance state name: ", err)
	}

	stateMap["name"] = nameString
	stateMap["code"] = state.Code

	stateReason := i.ec2Instance.StateReason
	stateMap["state_reason"] = awsauto.SafeStrp(stateReason.Code)
	stateMap[""] = awsauto.SafeStrp(stateReason.Message)
	return stateMap
}

// stateReasonMap converts the State Reason part of the ec2 struct into a friendlier map with 'reason' and 'code' fields.
func (i *ec2Instance) stateReasonMap() (stateReasonMap common.MapStr) {
	stateReasonMap = common.MapStr{}
	stateReason := i.ec2Instance.StateReason
	stateReasonMap["code"] = awsauto.SafeStrp(stateReason.Code)
	stateReasonMap["message"] = awsauto.SafeStrp(stateReason.Message)
	return stateReasonMap
}

// stateMap converts the State part of the ec2 struct into a friendlier map with 'reason' and 'code' fields.
func (i *ec2Instance) tagMap() (tagsMap []common.MapStr) {
	tags := i.ec2Instance.Tags
	tagsMap = []common.MapStr{}
	tagPair := common.MapStr{}
	for _, tag := range tags {
		tagPair["key"] = awsauto.SafeStrp(tag.Key)
		tagPair["value"] = awsauto.SafeStrp(tag.Value)
		tagsMap = append(tagsMap, tagPair)
	}

	return tagsMap
}
