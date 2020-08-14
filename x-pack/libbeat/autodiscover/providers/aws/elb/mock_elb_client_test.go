// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elb

import (
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/elasticloadbalancingv2iface"
)

func newMockELBClient(numResults int) mockELBClient {
	return mockELBClient{numResults: numResults}
}

type mockELBClient struct {
	elasticloadbalancingv2iface.ClientAPI
	numResults int
}
