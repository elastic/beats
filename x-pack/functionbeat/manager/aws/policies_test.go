// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"testing"

	"github.com/awslabs/goformation/v4/cloudformation"
	"github.com/awslabs/goformation/v4/cloudformation/iam"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/functionbeat/function/provider"
	fnaws "github.com/elastic/beats/v7/x-pack/functionbeat/provider/aws/aws"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestConfig(t *testing.T) {
	t.Run("test permissions for event_source_arn", testPolicies)
}

func testPolicies(t *testing.T) {
	cfg := conf.MustNewConfigFrom(map[string]interface{}{
		"name":        "myfunction",
		"description": "mydescription",
		"triggers": []map[string]interface{}{
			map[string]interface{}{
				"event_source_arn": "abc456",
			},
			map[string]interface{}{
				"event_source_arn": "abc1234",
			},
		},
	})

	k, err := fnaws.NewKinesis(&provider.DefaultProvider{}, cfg)
	if !assert.NoError(t, err) {
		return
	}

	i, ok := k.(installer)
	if !assert.True(t, ok) {
		return
	}

	policies := i.Policies()
	if !assert.Equal(t, 1, len(policies)) {
		return
	}

	// ensure permissions on specified resources
	expected := iam.Role_Policy{
		PolicyName: cloudformation.Join("-", []string{"fnb", "kinesis", "myfunction"}),
		PolicyDocument: map[string]interface{}{
			"Statement": []map[string]interface{}{
				map[string]interface{}{
					"Action": []string{
						"kinesis:GetRecords",
						"kinesis:GetShardIterator",
						"Kinesis:DescribeStream",
					},
					"Effect":   "Allow",
					"Resource": []string{"abc1234", "abc456"},
				},
			},
		},
	}

	assert.Equal(t, expected, policies[0])
}
