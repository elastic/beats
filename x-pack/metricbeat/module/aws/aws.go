// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"fmt"
	"strconv"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	resourcegroupstaggingapitypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/elastic/beats/v7/metricbeat/mb"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type describeRegionsClient interface {
	DescribeRegions(ctx context.Context, params *ec2.DescribeRegionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error)
}

// Config defines all required and optional parameters for aws metricsets
type Config struct {
	Period                time.Duration       `config:"period" validate:"nonzero,required"`
	DataGranularity       time.Duration       `config:"data_granularity"`
	Regions               []string            `config:"regions"`
	Latency               time.Duration       `config:"latency"`
	AWSConfig             awscommon.ConfigAWS `config:",inline"`
	TagsFilter            []Tag               `config:"tags_filter"`
	IncludeLinkedAccounts *bool               `config:"include_linked_accounts"`
}

// MetricSet is the base metricset for all aws metricsets
type MetricSet struct {
	mb.BaseMetricSet
	RegionsList           []string
	Endpoint              string
	Period                time.Duration
	DataGranularity       time.Duration
	Latency               time.Duration
	AwsConfig             *awssdk.Config
	MonitoringAccountName string
	MonitoringAccountID   string
	TagsFilter            []Tag
	IncludeLinkedAccounts bool
}

// Tag holds a configuration specific for ec2 and cloudwatch metricset.
type Tag struct {
	Key   string   `config:"key"`
	Value []string `config:"value"`
}

// ModuleName is the name of this module.
const ModuleName = "aws"

// IncludeLinkedAccountsDefault defines if we should include metrics from linked AWS accounts or not. Default is true.
// More information about cross-account Cloudwatch monitoring can be found at
// https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/Cross-Account-Cross-Region.html
const IncludeLinkedAccountsDefault = true

type LabelConstants struct {
	AccountIdIdx             int
	AccountLabelIdx          int
	MetricNameIdx            int
	NamespaceIdx             int
	StatisticIdx             int
	PeriodLabelIdx           int
	IdentifierNameIdx        int
	IdentifierValueIdx       int
	LabelLengthTotal         int
	LabelSeparator           string
	AccountLabel             string
	PeriodLabel              string
	BillingDimensionStartIdx int
}

var LabelConst = LabelConstants{
	AccountIdIdx:             0,
	AccountLabelIdx:          1,
	MetricNameIdx:            2,
	NamespaceIdx:             3,
	StatisticIdx:             4,
	PeriodLabelIdx:           5,
	IdentifierNameIdx:        6,
	IdentifierValueIdx:       7,
	LabelLengthTotal:         8,
	LabelSeparator:           "|",
	AccountLabel:             "${PROP('AccountLabel')}",
	PeriodLabel:              "${PROP('Period')}",
	BillingDimensionStartIdx: 3,
}

const CloudWatchPeriodName = "aws.cloudwatch.period"

func init() {
	if err := mb.Registry.AddModule(ModuleName, newModule); err != nil {
		panic(err)
	}
}

func newModule(base mb.BaseModule) (mb.Module, error) {
	var config Config
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}
	return &base, nil
}

// NewMetricSet creates a base metricset for aws metricsets
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	var config Config
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	awsConfig, err := awscommon.InitializeAWSConfig(config.AWSConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get aws credentials, please check AWS credential in config: %w", err)
	}

	ctx, cancel := getContextWithTimeout(DefaultApiTimeout)
	defer cancel()
	_, err = awsConfig.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve aws credentials, please check AWS credential in config: %w", err)
	}

	base.Logger().Debug("aws config endpoint = ", config.AWSConfig.Endpoint)
	if config.DataGranularity > config.Period {
		return nil, fmt.Errorf("Data Granularity cannot be larger than the period")
	}

	if config.DataGranularity == 0 {
		config.DataGranularity = config.Period
	}
	metricSet := MetricSet{
		BaseMetricSet:   base,
		Period:          config.Period,
		DataGranularity: config.DataGranularity,
		Latency:         config.Latency,
		AwsConfig:       &awsConfig,
		TagsFilter:      config.TagsFilter,
		Endpoint:        config.AWSConfig.Endpoint,
	}

	metricSet.IncludeLinkedAccounts = IncludeLinkedAccountsDefault
	if config.IncludeLinkedAccounts != nil {
		metricSet.IncludeLinkedAccounts = *config.IncludeLinkedAccounts
	}

	base.Logger().Debug("Metricset level config for period: ", metricSet.Period)
	base.Logger().Debug("Metricset level config for data granularity: ", metricSet.DataGranularity)
	base.Logger().Debug("Metricset level config for tags filter: ", metricSet.TagsFilter)
	base.Logger().Warn("extra charges on AWS API requests will be generated by this metricset")
	base.Logger().Debug("Metricset level config for including linked accounts: ", metricSet.IncludeLinkedAccounts)

	// If regions in config is not empty, then overwrite the awsConfig.Region
	if len(config.Regions) > 0 {
		awsConfig.Region = config.Regions[0]
	}

	// Get IAM account id
	svcSts := sts.NewFromConfig(awsConfig, func(o *sts.Options) {
		if config.AWSConfig.FIPSEnabled {
			o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
		}
	})
	ctx, cancel = getContextWithTimeout(DefaultApiTimeout)
	defer cancel()
	outputIdentity, err := svcSts.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		base.Logger().Warn("failed to get caller identity, please check permission setting: ", err)
	} else {
		metricSet.MonitoringAccountID = *outputIdentity.Account
		base.Logger().Debug("AWS Credentials belong to monitoring account ID: ", metricSet.MonitoringAccountID)
	}
	// Get account name/alias
	svcIam := iam.NewFromConfig(awsConfig, func(o *iam.Options) {
		if config.AWSConfig.FIPSEnabled {
			o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
		}

	})
	metricSet.MonitoringAccountName = getAccountName(svcIam, base, metricSet)

	// Construct MetricSet with a full regions list
	if config.Regions == nil {
		svcEC2 := ec2.NewFromConfig(awsConfig, func(o *ec2.Options) {
			if config.AWSConfig.FIPSEnabled {
				o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
			}
		})
		completeRegionsList, err := getRegions(svcEC2)
		if err != nil {
			return nil, err
		}

		metricSet.RegionsList = completeRegionsList
		base.Logger().Debug("Metricset level config for regions: ", metricSet.RegionsList)
		return &metricSet, nil
	}

	// Construct MetricSet with specific regions list from config
	metricSet.RegionsList = config.Regions
	base.Logger().Debug("Metricset level config for regions: ", metricSet.RegionsList)
	return &metricSet, nil
}

func getRegions(svc describeRegionsClient) ([]string, error) {
	completeRegionsList := make([]string, 0)
	input := &ec2.DescribeRegionsInput{}
	output, err := svc.DescribeRegions(context.TODO(), input)
	if err != nil {
		err = fmt.Errorf("failed DescribeRegions: %w", err)
		return completeRegionsList, err
	}
	for _, region := range output.Regions {
		completeRegionsList = append(completeRegionsList, *region.RegionName)
	}
	return completeRegionsList, err
}

func getAccountName(svc *iam.Client, base mb.BaseMetricSet, metricSet MetricSet) string {
	ctx, cancel := getContextWithTimeout(DefaultApiTimeout)
	defer cancel()
	output, err := svc.ListAccountAliases(ctx, &iam.ListAccountAliasesInput{})

	accountName := metricSet.MonitoringAccountID
	if err != nil {
		base.Logger().Warn("failed to list account aliases, please check permission setting: ", err)
		return accountName
	}

	// When there is no account alias, account ID will be used as cloud.account.name
	if len(output.AccountAliases) == 0 {
		accountName = metricSet.MonitoringAccountID
		base.Logger().Debug("AWS Credentials belong to account ID: ", metricSet.MonitoringAccountID)
		return accountName
	}

	// There can be more than one aliases for each account, for now we are only
	// collecting the first one.
	accountName = output.AccountAliases[0]
	base.Logger().Debug("AWS Credentials belong to account name: ", metricSet.MonitoringAccountName)
	return accountName
}

// StringInSlice checks if a string is already exists in list and its location
func StringInSlice(str string, list []string) (bool, int) {
	for idx, v := range list {
		if v == str {
			return true, idx
		}
	}
	// If this string doesn't exist in given list, then return location to be -1
	return false, -1
}

// InitEvent initialize mb.Event with basic information like service.name, cloud.provider
func InitEvent(regionName string, accountName string, accountID string, timestamp time.Time, periodLabel string) mb.Event {
	event := mb.Event{
		Timestamp:       timestamp,
		MetricSetFields: mapstr.M{},
		ModuleFields:    mapstr.M{},
		RootFields:      mapstr.M{},
	}

	period, err := strconv.Atoi(periodLabel)
	if err == nil {
		_, _ = event.RootFields.Put(CloudWatchPeriodName, period)
	}
	_, _ = event.RootFields.Put("cloud.provider", "aws")
	if regionName != "" {
		_, _ = event.RootFields.Put("cloud.region", regionName)
	}
	if accountName != "" {
		_, _ = event.RootFields.Put("cloud.account.name", accountName)
	}
	if accountID != "" {
		_, _ = event.RootFields.Put("cloud.account.id", accountID)
	}
	return event
}

// CheckTagFiltersExist compare tags filter with a set of tags to see if tags
// filter is a subset of tags
func CheckTagFiltersExist(tagsFilter []Tag, tags interface{}) bool {
	var tagKeys []string
	var tagValues []string

	switch tags := tags.(type) {
	case []resourcegroupstaggingapitypes.Tag:
		for _, tag := range tags {
			tagKeys = append(tagKeys, *tag.Key)
			tagValues = append(tagValues, *tag.Value)
		}
	case []ec2types.Tag:
		for _, tag := range tags {
			tagKeys = append(tagKeys, *tag.Key)
			tagValues = append(tagValues, *tag.Value)
		}
	case []rdstypes.Tag:
		for _, tag := range tags {
			tagKeys = append(tagKeys, *tag.Key)
			tagValues = append(tagValues, *tag.Value)
		}
	}

	for _, tagFilter := range tagsFilter {
		if exists, idx := StringInSlice(tagFilter.Key, tagKeys); exists {
			valueExists, _ := StringInSlice(tagValues[idx], tagFilter.Value)
			if !valueExists {
				return false
			}
		} else {
			return false
		}
	}
	return true
}
