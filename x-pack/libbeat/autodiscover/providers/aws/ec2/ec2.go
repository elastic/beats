// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	awsauto "github.com/elastic/beats/x-pack/libbeat/autodiscover/providers/aws"
)

type ec2Instance struct {
	ec2Instance ec2.Instance
}

// toMap converts this ec2Instance into the form consumed as metadata in the autodiscovery process.
func (i *ec2Instance) toMap() common.MapStr {
	instanceType, err := i.ec2Instance.InstanceType.MarshalValue()
	if err != nil {
		logp.Error(errors.Wrap(err, "MarshalValue failed for instance type: "))
	}

	monitoringState, err := i.ec2Instance.Monitoring.State.MarshalValue()
	if err != nil {
		logp.Error(errors.Wrap(err, "MarshalValue failed for monitoring state: "))
	}

	architecture, err := i.ec2Instance.Architecture.MarshalValue()
	if err != nil {
		logp.Error(errors.Wrap(err, "MarshalValue failed for architecture: "))
	}

	m := common.MapStr{
		"instance_id":      awsauto.SafeStrp(i.ec2Instance.InstanceId),
		"image_id":         awsauto.SafeStrp(i.ec2Instance.ImageId),
		"vpc_id":           awsauto.SafeStrp(i.ec2Instance.VpcId),
		"subnet_id":        awsauto.SafeStrp(i.ec2Instance.SubnetId),
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
	}

	for _, tag := range i.ec2Instance.Tags {
		m.Put("tags."+awsauto.SafeStrp(tag.Key), awsauto.SafeStrp(tag.Value))
	}
	return m
}

func (i *ec2Instance) toCloudMap() common.MapStr {
	m := common.MapStr{}
	availabilityZone := awsauto.SafeStrp(i.ec2Instance.Placement.AvailabilityZone)
	m["availability_zone"] = availabilityZone
	m["provider"] = "aws"

	// The region is just an AZ with the last character removed
	m["region"] = availabilityZone[:len(availabilityZone)-1]
	return m
}

// stateMap converts the State part of the ec2 struct into a friendlier map with 'reason' and 'code' fields.
func (i *ec2Instance) stateMap() (stateMap common.MapStr) {
	state := i.ec2Instance.State
	stateMap = common.MapStr{}
	nameString, err := state.Name.MarshalValue()
	if err != nil {
		logp.Error(errors.Wrap(err, "MarshalValue failed for instance state name: "))
	}

	stateMap["name"] = nameString
	stateMap["code"] = state.Code
	return stateMap
}
