// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elb

import (
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_newAPIFetcher(t *testing.T) {
	client := newMockELBClient(0)
	fetcher := newAPIFetcher([]elasticloadbalancingv2.DescribeLoadBalancersAPIClient{client})
	require.NotNil(t, fetcher)
}
