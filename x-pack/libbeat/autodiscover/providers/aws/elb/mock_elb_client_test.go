// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elb

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elasticloadbalancingv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/aws/smithy-go/middleware"
)

func newMockELBClient(numResults int) autodiscoverElbClient {
	return &mockELBClient{numResults: numResults}
}

type mockELBClient struct {
	elasticloadbalancingv2.Client
	numResults int
}

func (m *mockELBClient) DescribeLoadBalancers(context.Context, *elasticloadbalancingv2.DescribeLoadBalancersInput, ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error) {
	emptyString := ""
	return &elasticloadbalancingv2.DescribeLoadBalancersOutput{
		LoadBalancers:  []elasticloadbalancingv2types.LoadBalancer{},
		NextMarker:     &emptyString,
		ResultMetadata: middleware.Metadata{},
	}, nil
}
