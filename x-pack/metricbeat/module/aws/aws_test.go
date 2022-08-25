// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package aws

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	resourcegroupstaggingapitypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/stretchr/testify/assert"
)

type mockProvider struct {
	listAccountAliasesTests  func(context.Context, *iam.ListAccountAliasesInput, ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error)
	describeRegionsTests     func(context.Context, *ec2.DescribeRegionsInput, ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error)
	newStsFromConfigTests    func(awssdk.Config, ...func(*sts.Options)) *sts.Client
	getCallerIdentityTests   func(*sts.Client, context.Context, *sts.GetCallerIdentityInput, ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
	newIamFromConfigTests    func(awssdk.Config, ...func(*iam.Options)) iam.ListAccountAliasesAPIClient
	retrieveCredentialsTests func(context.Context) (awssdk.Credentials, error)
	*ec2.Client
}

func (m *mockProvider) Retrieve(ctx context.Context) (awssdk.Credentials, error) {
	return m.retrieveCredentialsTests(ctx)
}

func (m *mockProvider) newIamFromConfig(config awssdk.Config, f ...func(*iam.Options)) iam.ListAccountAliasesAPIClient {
	return m.newIamFromConfigTests(config, f...)
}

func (m *mockProvider) newStsFromConfig(config awssdk.Config, f ...func(*sts.Options)) *sts.Client {
	return m.newStsFromConfigTests(config, f...)
}

func (m *mockProvider) getCallerIdentity(client *sts.Client, ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	return m.getCallerIdentityTests(client, ctx, params, optFns...)
}

func (m *mockProvider) ListAccountAliases(ctx context.Context, input *iam.ListAccountAliasesInput, f ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error) {
	return m.listAccountAliasesTests(ctx, input, f...)
}

func (m *mockProvider) DescribeRegions(ctx context.Context, params *ec2.DescribeRegionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error) {
	return m.describeRegionsTests(ctx, params, optFns...)
}

func TestGetRegions(t *testing.T) {
	mockSvc := &mockProvider{
		describeRegionsTests: func(ctx context.Context, params *ec2.DescribeRegionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error) {
			return &ec2.DescribeRegionsOutput{
				Regions: []ec2types.Region{{RegionName: awssdk.String("us-west-1")}}}, nil
		},
	}

	regionsList, err := getRegions(mockSvc)
	if err != nil {
		t.Fatalf("failed getRegions: %s", err)
	}
	assert.Equal(t, 1, len(regionsList))
	assert.Equal(t, "us-west-1", regionsList[0])

	mockSvc.describeRegionsTests = func(ctx context.Context, params *ec2.DescribeRegionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error) {
		return nil, errors.New("error getting regions")
	}
	regionsList, err = getRegions(mockSvc)
	assert.Error(t, err)
	assert.Equal(t, []string{}, regionsList)
	assert.ErrorContains(t, err, "failed DescribeRegions")

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
)

func TestCheckTagFiltersExist(t *testing.T) {
	cases := []struct {
		title          string
		tagFilters     []Tag
		tags           interface{}
		expectedExists bool
	}{
		{
			"tagFilters are included in rds tags",
			[]Tag{
				{
					Key:   "Name",
					Value: "RDS",
				},
				{
					Key:   "Organization",
					Value: "Engineering",
				},
			},
			[]rdstypes.Tag{
				{
					Key:   awssdk.String(tagKey1),
					Value: awssdk.String("RDS"),
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
			"tagFilters are included in ec2 tags",
			[]Tag{
				{
					Key:   "Name",
					Value: "ECS Instance",
				},
				{
					Key:   "Organization",
					Value: "Engineering",
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
					Value: "test",
				},
				{
					Key:   "Organization",
					Value: "Engineering",
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
					Value: "test",
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
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			exists := CheckTagFiltersExist(c.tagFilters, c.tags)
			assert.Equal(t, c.expectedExists, exists)
		})
	}
}

func TestInitEvent(t *testing.T) {
	regionName := "myRegion"
	accountName := "myAccountName"
	accountID := "myAccountId"
	timestamp := time.Now()

	//Check hardcoded expected data
	assertEqual := func(mbEvent mb.Event, expected interface{}, key string) {
		val, err := mbEvent.RootFields.GetValue(key)
		if err != nil {
			t.Fail()
		}
		assert.Equal(t, expected, val)
	}
	mbEvent1 := InitEvent(regionName, accountName, accountID, timestamp)
	assertEqual(mbEvent1, "aws", "cloud.provider")
	assertEqual(mbEvent1, regionName, "cloud.region")
	assertEqual(mbEvent1, accountName, "cloud.account.name")
	assertEqual(mbEvent1, accountID, "cloud.account.id")

	// Leave every field empty
	mbEvent2 := InitEvent("", "", "", time.Now())
	assertNil := func(mbEvent mb.Event, key string) {
		val, err := mbEvent.RootFields.GetValue(key)
		if err == nil {
			t.Fail()
		}
		assert.Nil(t, val)
	}
	assertEqual(mbEvent1, "aws", "cloud.provider")
	assertNil(mbEvent2, "cloud.region")
	assertNil(mbEvent2, "cloud.account.id")
	assertNil(mbEvent2, "cloud.account.name")
}

func TestGetAccountName(t *testing.T) {
	mock := mockProvider{
		listAccountAliasesTests: func(ctx context.Context, input *iam.ListAccountAliasesInput, f ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error) {
			return &iam.ListAccountAliasesOutput{AccountAliases: []string{"alias1", "alias2"}}, nil
		},
	}
	logger := logp.NewLogger("aws-testing")
	name := getAccountName(&mock, logger, "myAccountID")
	assert.Equal(t, "alias1", name)

	mock.listAccountAliasesTests = func(ctx context.Context, input *iam.ListAccountAliasesInput, f ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error) {
		return &iam.ListAccountAliasesOutput{AccountAliases: []string{}}, nil
	}
	name = getAccountName(&mock, logger, "myAccountID")
	assert.Equal(t, "myAccountID", name)

	mock.listAccountAliasesTests = func(ctx context.Context, input *iam.ListAccountAliasesInput, f ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error) {
		return nil, errors.New("an error")
	}
	name = getAccountName(&mock, logger, "myAccountID")
	assert.Equal(t, "myAccountID", name)
}

func TestSetupMetricset(t *testing.T) {
	log := logp.NewLogger("aws-testing")
	mock := &mockProvider{
		listAccountAliasesTests: func(ctx context.Context, input *iam.ListAccountAliasesInput, f ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error) {
			return &iam.ListAccountAliasesOutput{AccountAliases: []string{"alias1", "alias2"}}, nil
		},
		newStsFromConfigTests: func(config awssdk.Config, f ...func(*sts.Options)) *sts.Client {
			return nil
		},
		getCallerIdentityTests: func(client *sts.Client, ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
			return nil, nil
		},
	}
	mock.newIamFromConfigTests = func(config awssdk.Config, f ...func(*iam.Options)) iam.ListAccountAliasesAPIClient {
		return mock
	}

	config := Config{
		Period:     0,
		Regions:    nil,
		Latency:    0,
		AWSConfig:  awscommon.ConfigAWS{},
		TagsFilter: nil,
	}
	awsConfig := awssdk.Config{}

	t.Run("no region specified", func(t *testing.T) {
		m := newMetricset(mb.BaseMetricSet{}, config, awsConfig)
		err := m.setupMetricset(log, config, mock)
		assert.Error(t, err, "failed DescribeRegions: operation error EC2: DescribeRegions, failed to resolve service endpoint, an AWS region is required, but was not found")
	})

	t.Run("region specified", func(t *testing.T) {
		awsConfig.Region = "a_region"
		config.Regions = []string{awsConfig.Region, "another_region"}
		m := newMetricset(mb.BaseMetricSet{}, config, awsConfig)
		err := m.setupMetricset(log, config, mock)
		assert.NoError(t, err)
	})
}
