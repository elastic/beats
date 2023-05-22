// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package aws

import (
	"context"
	"testing"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	resourcegroupstaggingapitypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/stretchr/testify/assert"
)

// MockEC2Client struct is used for unit tests.
type MockEC2Client struct {
	*ec2.Client
}

func (m *MockEC2Client) DescribeRegions(ctx context.Context, params *ec2.DescribeRegionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error) {
	return &ec2.DescribeRegionsOutput{
		Regions: []ec2types.Region{
			{
				RegionName: awssdk.String("us-west-1"),
			},
		},
	}, nil
}

func TestGetRegions(t *testing.T) {
	mockSvc := &MockEC2Client{}
	regionsList, err := getRegions(mockSvc)
	if err != nil {
		t.Fatalf("failed getRegions: %s", err)
	}
	assert.Equal(t, 1, len(regionsList))
	assert.Equal(t, "us-west-1", regionsList[0])
}

func TestStringInSlice(t *testing.T) {
	cases := []struct {
		target         string
		slice          []string
		expectedExists bool
		expectedIdx    int
	}{
		{
			"bar",
			[]string{"foo", "bar", "baz"},
			true,
			1,
		},
		{
			"test",
			[]string{"foo", "bar", "baz"},
			false,
			-1,
		},
	}
	for _, c := range cases {
		exists, idx := StringInSlice(c.target, c.slice)
		assert.Equal(t, c.expectedExists, exists)
		assert.Equal(t, c.expectedIdx, idx)
	}
}

var (
	tagKey1   = "Name"
	tagValue1 = "ECS Instance"
	tagKey2   = "User"
	tagValue2 = "foobar"
	tagKey3   = "Organization"
	tagValue3 = "Engineering"
	tagValue4 = "Product"
	tagValue5 = "ElastiCache Redis"
)

func TestCheckTagFiltersExist(t *testing.T) {
	cases := []struct {
		title          string
		tagFilters     []Tag
		tags           interface{}
		expectedExists bool
	}{
		{
			"tagFilters are included in ec2 tags",
			[]Tag{
				{
					Key:   "Name",
					Value: []string{"ECS Instance"},
				},
				{
					Key:   "Organization",
					Value: []string{"Engineering"},
				},
			},
			[]ec2types.Tag{
				{
					Key:   awssdk.String(tagKey1),
					Value: awssdk.String(tagValue1),
				},
				{
					Key:   awssdk.String(tagKey2),
					Value: awssdk.String(tagValue2),
				},
				{
					Key:   awssdk.String(tagKey3),
					Value: awssdk.String(tagValue3),
				},
			},
			true,
		},
		{
			"one set of tagFilters is included in resourcegroupstaggingapi tags",
			[]Tag{
				{
					Key:   "Name",
					Value: []string{"test"},
				},
				{
					Key:   "Organization",
					Value: []string{"Engineering"},
				},
			},
			[]resourcegroupstaggingapitypes.Tag{
				{
					Key:   awssdk.String(tagKey1),
					Value: awssdk.String(tagValue1),
				},
				{
					Key:   awssdk.String(tagKey2),
					Value: awssdk.String(tagValue2),
				},
				{
					Key:   awssdk.String(tagKey3),
					Value: awssdk.String(tagValue3),
				},
			},
			false,
		},
		{
			"tagFilters is not included in resourcegroupstaggingapi tags",
			[]Tag{
				{
					Key:   "Name",
					Value: []string{"test"},
				},
			},
			[]resourcegroupstaggingapitypes.Tag{
				{
					Key:   awssdk.String(tagKey1),
					Value: awssdk.String(tagValue1),
				},
				{
					Key:   awssdk.String(tagKey2),
					Value: awssdk.String(tagValue2),
				},
				{
					Key:   awssdk.String(tagKey3),
					Value: awssdk.String(tagValue3),
				},
			},
			false,
		},
		{
			"more than one value per key of tagFilters is included in resourcegroupstaggingapi tags",
			[]Tag{
				{
					Key:   "Organization",
					Value: []string{"Product", "Engineering"},
				},
			},
			[][]resourcegroupstaggingapitypes.Tag{
				{
					{
						Key:   awssdk.String(tagKey1),
						Value: awssdk.String(tagValue1),
					},
					{
						Key:   awssdk.String(tagKey3),
						Value: awssdk.String(tagValue3),
					},
				},
				{
					{
						Key:   awssdk.String(tagKey2),
						Value: awssdk.String(tagValue2),
					},
					{
						Key:   awssdk.String(tagKey3),
						Value: awssdk.String(tagValue4),
					},
				},
			},
			true,
		},
		{
			"a set of tagFilters where every key contains more than one value is included in resourcegroupstaggingapi tags",
			[]Tag{
				{
					Key:   "Name",
					Value: []string{"ECS Instance", "ElastiCache Redis"},
				},
				{
					Key:   "Organization",
					Value: []string{"Product", "Engineering"},
				},
			},
			[][]resourcegroupstaggingapitypes.Tag{
				{
					{
						Key:   awssdk.String(tagKey1),
						Value: awssdk.String(tagValue1),
					},
					{
						Key:   awssdk.String(tagKey3),
						Value: awssdk.String(tagValue3),
					},
				},
				{
					{
						Key:   awssdk.String(tagKey1),
						Value: awssdk.String(tagValue5),
					},
					{
						Key:   awssdk.String(tagKey3),
						Value: awssdk.String(tagValue4),
					},
				},
			},
			true,
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			switch c.tags.(type) {
			case [][]resourcegroupstaggingapitypes.Tag:
				allExist := true
				for _, tags := range c.tags.([][]resourcegroupstaggingapitypes.Tag) {
					exists := CheckTagFiltersExist(c.tagFilters, tags)
					allExist = exists && allExist
				}
				assert.Equal(t, c.expectedExists, allExist)
			default:
				exists := CheckTagFiltersExist(c.tagFilters, c.tags)
				assert.Equal(t, c.expectedExists, exists)
			}
		})
	}
}
