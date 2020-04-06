// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// mockFetcher is a fetcher that returns a customizable list of results, useful for testing.
type mockFetcher struct {
	ec2Instances []*ec2Instance
	err          error
	lock         sync.Mutex
}

func newMockFetcher(lbListeners []*ec2Instance, err error) *mockFetcher {
	return &mockFetcher{ec2Instances: lbListeners, err: err}
}

func (f *mockFetcher) fetch(ctx context.Context) ([]*ec2Instance, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	result := make([]*ec2Instance, len(f.ec2Instances))
	copy(result, f.ec2Instances)

	return result, f.err
}

func (f *mockFetcher) setEC2s(newEC2s []*ec2Instance) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.ec2Instances = newEC2s
}

func (f *mockFetcher) setError(err error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.ec2Instances = []*ec2Instance{}
	f.err = err
}

func fakeEC2Instance() *ec2Instance {
	runningCode := int64(16)
	coreCount := int64(1)
	threadsPerCore := int64(1)
	publicDNSName := "ec2-1-2-3-4.us-west-1.compute.amazonaws.com"
	publicIP := "1.2.3.4"
	privateDNSName := "ip-5-6-7-8.us-west-1.compute.internal"
	privateIP := "5.6.7.8"
	instanceID := "i-123"

	instance := ec2.Instance{
		InstanceId:   aws.String(instanceID),
		InstanceType: ec2.InstanceTypeT2Medium,
		Placement: &ec2.Placement{
			AvailabilityZone: aws.String("us-west-1a"),
		},
		ImageId: aws.String("image-123"),
		State: &ec2.InstanceState{
			Name: ec2.InstanceStateNameRunning,
			Code: &runningCode,
		},
		Monitoring: &ec2.Monitoring{
			State: ec2.MonitoringStateDisabled,
		},
		CpuOptions: &ec2.CpuOptions{
			CoreCount:      &coreCount,
			ThreadsPerCore: &threadsPerCore,
		},
		PublicDnsName:    &publicDNSName,
		PublicIpAddress:  &publicIP,
		PrivateDnsName:   &privateDNSName,
		PrivateIpAddress: &privateIP,
	}
	return &ec2Instance{ec2Instance: instance}
}
