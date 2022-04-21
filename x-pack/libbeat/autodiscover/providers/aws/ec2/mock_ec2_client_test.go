// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func newMockEC2Client(numResults int) ec2.DescribeInstancesAPIClient {
	return &mockEC2Client{numResults: numResults}
}

type mockEC2Client struct {
	numResults int
}

func (m *mockEC2Client)DescribeInstances(context.Context, *ec2.DescribeInstancesInput, ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error){
	return nil, fmt.Errorf("not implemented")
}
