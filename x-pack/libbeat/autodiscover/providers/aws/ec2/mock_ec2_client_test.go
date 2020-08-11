// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
)

func newMockEC2Client(numResults int) mockEC2Client {
	return mockEC2Client{numResults: numResults}
}

type mockEC2Client struct {
	ec2iface.ClientAPI
	numResults int
}
