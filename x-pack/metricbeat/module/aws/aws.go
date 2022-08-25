// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"fmt"
	"github.com/elastic/elastic-agent-libs/logp"

	"strings"
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
	Period     time.Duration       `config:"period" validate:"nonzero,required"`
	Regions    []string            `config:"regions"`
	Latency    time.Duration       `config:"latency"`
	AWSConfig  awscommon.ConfigAWS `config:",inline"`
	TagsFilter []Tag               `config:"tags_filter"`
}

// MetricSet is the base metricset for all aws metricsets
type MetricSet struct {
	mb.BaseMetricSet
	RegionsList []string
	Endpoint    string
	Period      time.Duration
	Latency     time.Duration
	AwsConfig   awssdk.Config
	AccountName string
	AccountID   string
	TagsFilter  []Tag
}

// Tag holds a configuration specific for ec2 and cloudwatch metricset.
type Tag struct {
	Key   string `config:"key"`
	Value string `config:"value"`
}

// ModuleName is the name of this module.
const ModuleName = "aws"

func init() {
	if err := mb.Registry.AddModule(ModuleName, newModule); err != nil {
		panic(err)
	}
}

// NewMetricSet creates a base metricset for aws metricsets
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	var config Config
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	awsConfig, err := setupCredentials(config)
	if err != nil {
		return nil, fmt.Errorf("could not setup credentials: '%v'", err)
	}

	metricSet := newMetricset(base, config, awsConfig)

	if err = metricSet.setupMetricset(metricSet.Logger(), config, &stsMediatorAws{}); err != nil {
		return nil, fmt.Errorf("could not initialize AWS metricset: '%v'", err)
	}

	return metricSet, nil
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
func InitEvent(regionName string, accountName string, accountID string, timestamp time.Time) mb.Event {
	event := mb.Event{
		Timestamp:       timestamp,
		MetricSetFields: mapstr.M{},
		ModuleFields:    mapstr.M{},
		RootFields:      mapstr.M{},
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
		if exists, idx := StringInSlice(tagFilter.Key, tagKeys); !exists || tagValues[idx] != tagFilter.Value {
			return false
		}
	}
	return true
}

type awsSetupMediator interface {
	newStsFromConfig(awssdk.Config, ...func(*sts.Options)) *sts.Client
	newIamFromConfig(awssdk.Config, ...func(*iam.Options)) iam.ListAccountAliasesAPIClient
	getCallerIdentity(*sts.Client, context.Context, *sts.GetCallerIdentityInput, ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

func newModule(base mb.BaseModule) (mb.Module, error) {
	var config Config
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}
	return &base, nil
}

//setupCredentials uses the libbeat AWS common package to retrieve a set of credentials
func setupCredentials(config Config) (awssdk.Config, error) {
	awsConfig, err := awscommon.InitializeAWSConfig(config.AWSConfig)
	if err != nil {
		return awssdk.Config{}, fmt.Errorf("failed to get aws credentials, please check AWS credential in config: %w", err)
	}

	_, err = awsConfig.Credentials.Retrieve(context.Background())
	if err != nil {
		return awssdk.Config{}, fmt.Errorf("failed to retrieve aws credentials, please check AWS credential in config: %w", err)
	}

	return awsConfig, err
}

func newMetricset(base mb.BaseMetricSet, config Config, awsConfig awssdk.Config) *MetricSet {
	return &MetricSet{
		BaseMetricSet: base,
		Period:        config.Period,
		Latency:       config.Latency,
		AwsConfig:     awsConfig,
		TagsFilter:    config.TagsFilter,
		Endpoint:      config.AWSConfig.Endpoint,
	}
}

type stsMediatorAws struct{}

func (s *stsMediatorAws) newIamFromConfig(cfg awssdk.Config, optFns ...func(*iam.Options)) iam.ListAccountAliasesAPIClient {
	return iam.NewFromConfig(cfg, optFns...)
}

func (s *stsMediatorAws) getCallerIdentity(client *sts.Client, ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	return client.GetCallerIdentity(ctx, params, optFns...)
}

func (s *stsMediatorAws) newStsFromConfig(config awssdk.Config, optFns ...func(*sts.Options)) *sts.Client {
	return sts.NewFromConfig(config, optFns...)
}

func (m *MetricSet) setupMetricset(logger *logp.Logger, config Config, awsSetup awsSetupMediator) error {
	logger.Debug("aws config endpoint = ", config.AWSConfig.Endpoint)
	logger.Debug("Metricset level config for period: ", m.Period)
	logger.Debug("Metricset level config for tags filter: ", m.TagsFilter)
	logger.Warn("extra charges on AWS API requests will be generated by this metricset")

	// If regions in config is not empty, then overwrite the awsConfig.Region
	if len(config.Regions) > 0 {
		logger.Debug("using only region '%s'. Omitting '%v'", config.Regions[0], config.Regions[1:])
		m.AwsConfig.Region = config.Regions[0]
	}

	// Get IAM account id
	stsServiceName := awscommon.CreateServiceName("sts", config.AWSConfig.FIPSEnabled, m.AwsConfig.Region)
	awsConfigEnriched := awscommon.EnrichAWSConfigWithEndpoint(config.AWSConfig.Endpoint, stsServiceName, "", m.AwsConfig)
	svcSts := awsSetup.newStsFromConfig(awsConfigEnriched)

	outputIdentity, err := awsSetup.getCallerIdentity(svcSts, context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		logger.Warn("failed to get caller identity, please check permission setting: ", err)
	} else if outputIdentity == nil || outputIdentity.Account == nil {
		logger.Warn("expected account id not found, please check permission setting")
	} else {
		m.AccountID = *outputIdentity.Account
		logger.Debug("AWS Credentials belong to account ID: ", m.AccountID)
	}

	// Get account name/alias
	iamRegion := ""
	if strings.HasPrefix(m.AwsConfig.Region, "us-gov-") {
		iamRegion = "us-gov"
	}
	iamServiceName := awscommon.CreateServiceName("iam", config.AWSConfig.FIPSEnabled, m.AwsConfig.Region)
	configEnriched := awscommon.EnrichAWSConfigWithEndpoint(config.AWSConfig.Endpoint, iamServiceName, iamRegion, m.AwsConfig)
	svcIam := awsSetup.newIamFromConfig(configEnriched)
	m.AccountName = getAccountName(svcIam, logger, m.AccountID)

	// Construct MetricSet with a full regions list
	if config.Regions == nil {
		ec2ServiceName := awscommon.CreateServiceName("ec2", config.AWSConfig.FIPSEnabled, m.AwsConfig.Region)
		awsConfigEnriched := awscommon.EnrichAWSConfigWithEndpoint(config.AWSConfig.Endpoint, ec2ServiceName, "", m.AwsConfig)
		svcEC2 := ec2.NewFromConfig(awsConfigEnriched)
		completeRegionsList, err := getRegions(svcEC2)
		if err != nil {
			return err
		}

		m.RegionsList = completeRegionsList
		logger.Debug("Metricset level config for regions: ", m.RegionsList)
		return nil
	}

	// Construct MetricSet with specific regions list from config
	m.RegionsList = config.Regions
	logger.Debug("Metricset level config for regions: ", m.RegionsList)

	return nil
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

func getAccountName(svc iam.ListAccountAliasesAPIClient, logger *logp.Logger, accountID string) string {
	output, err := svc.ListAccountAliases(context.TODO(), &iam.ListAccountAliasesInput{})

	accountName := accountID
	if err != nil {
		logger.Warn("failed to list account aliases, please check permission setting: ", err)
		return accountName
	}

	// When there is no account alias, account ID will be used as cloud.account.name
	if len(output.AccountAliases) == 0 {
		accountName = accountID
		logger.Debug("AWS Credentials belong to account ID: ", accountID)
		return accountName
	}

	// There can be more than one aliases for each account, for now we are only
	// collecting the first one.
	accountName = output.AccountAliases[0]
	logger.Debug("AWS Credentials belong to account name: ", accountName)
	return accountName
}
