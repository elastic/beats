// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
	awsauto "github.com/menderesk/beats/v7/x-pack/libbeat/autodiscover/providers/aws"
)

type ec2Instance struct {
	ec2Instance ec2.Instance
}

// toMap converts this ec2Instance into the form consumed as metadata in the autodiscovery process.
func (i *ec2Instance) toMap() common.MapStr {
	architecture, err := i.ec2Instance.Architecture.MarshalValue()
	if err != nil {
		logp.Error(errors.Wrap(err, "MarshalValue failed for architecture: "))
	}

	m := common.MapStr{
		"image":            i.toImage(),
		"vpc":              i.toVpc(),
		"subnet":           i.toSubnet(),
		"private":          i.toPrivate(),
		"public":           i.toPublic(),
		"monitoring":       i.toMonitoringState(),
		"kernel":           i.toKernel(),
		"state":            i.stateMap(),
		"architecture":     architecture,
		"root_device_name": awsauto.SafeString(i.ec2Instance.RootDeviceName),
	}

	for _, tag := range i.ec2Instance.Tags {
		m.Put("tags."+awsauto.SafeString(tag.Key), awsauto.SafeString(tag.Value))
	}
	return m
}

func (i *ec2Instance) instanceID() string {
	return awsauto.SafeString(i.ec2Instance.InstanceId)
}

func (i *ec2Instance) toImage() common.MapStr {
	m := common.MapStr{}
	m["id"] = awsauto.SafeString(i.ec2Instance.ImageId)
	return m
}

func (i *ec2Instance) toMonitoringState() common.MapStr {
	monitoringState, err := i.ec2Instance.Monitoring.State.MarshalValue()
	if err != nil {
		logp.Error(errors.Wrap(err, "MarshalValue failed for monitoring state: "))
	}

	m := common.MapStr{}
	m["state"] = monitoringState
	return m
}

func (i *ec2Instance) toPrivate() common.MapStr {
	m := common.MapStr{}
	m["ip"] = awsauto.SafeString(i.ec2Instance.PrivateIpAddress)
	m["dns_name"] = awsauto.SafeString(i.ec2Instance.PrivateDnsName)
	return m
}

func (i *ec2Instance) toPublic() common.MapStr {
	m := common.MapStr{}
	m["ip"] = awsauto.SafeString(i.ec2Instance.PublicIpAddress)
	m["dns_name"] = awsauto.SafeString(i.ec2Instance.PublicDnsName)
	return m
}

func (i *ec2Instance) toVpc() common.MapStr {
	m := common.MapStr{}
	m["id"] = awsauto.SafeString(i.ec2Instance.VpcId)
	return m
}

func (i *ec2Instance) toSubnet() common.MapStr {
	m := common.MapStr{}
	m["id"] = awsauto.SafeString(i.ec2Instance.SubnetId)
	return m
}

func (i *ec2Instance) toKernel() common.MapStr {
	m := common.MapStr{}
	m["id"] = awsauto.SafeString(i.ec2Instance.KernelId)
	return m
}

func (i *ec2Instance) toCloudMap() common.MapStr {
	m := common.MapStr{}
	availabilityZone := awsauto.SafeString(i.ec2Instance.Placement.AvailabilityZone)
	m["availability_zone"] = availabilityZone
	m["provider"] = "aws"

	// The region is just an AZ with the last character removed
	m["region"] = availabilityZone[:len(availabilityZone)-1]

	instance := common.MapStr{}
	instance["id"] = i.instanceID()
	m["instance"] = instance

	instanceType, err := i.ec2Instance.InstanceType.MarshalValue()
	if err != nil {
		logp.Error(errors.Wrap(err, "MarshalValue failed for instance type: "))
	}
	machine := common.MapStr{}
	machine["type"] = instanceType
	m["machine"] = machine
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
