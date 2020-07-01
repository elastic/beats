// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindServiceNameFromARN(t *testing.T) {
	cases := []struct {
		arn                 string
		expectedServiceName string
	}{
		{
			"arn:aws:ec2:us-west-1:428152502467:instance/i-0db991e92aba79a3c",
			"ec2",
		},
	}

	for _, c := range cases {
		serviceName, err := findServiceNameFromARN(c.arn)
		assert.NoError(t, err)
		assert.Equal(t, c.expectedServiceName, serviceName)
	}
}
