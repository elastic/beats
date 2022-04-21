// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_newAPIFetcher(t *testing.T) {
	client := newMockEC2Client(0)
	fetcher := newAPIFetcher([]ec2.DescribeInstancesAPIClient{client})
	require.NotNil(t, fetcher)
}
