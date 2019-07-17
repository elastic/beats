// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"time"

	"github.com/elastic/beats/libbeat/common"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/metricbeat/mb"
)

// Config defines all required and optional parameters for aws metricsets
type Config struct {
	Period          time.Duration `config:"period" validate:"nonzero,required"`
	AccessKeyID     string        `config:"access_key_id"`
	SecretAccessKey string        `config:"secret_access_key"`
	SessionToken    string        `config:"session_token"`
	DefaultRegion   string        `config:"default_region"`
	Regions         []string      `config:"regions"`
}

// MetricSet is the base metricset for all aws metricsets
type MetricSet struct {
	mb.BaseMetricSet
	RegionsList []string
	Period      time.Duration
	AwsConfig   *awssdk.Config
}

// ModuleName is the name of this module.
const ModuleName = "aws"

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

	cfg := defaults.Config()
	awsCredentials := awssdk.Credentials{}
	if config.AccessKeyID != "" && config.SecretAccessKey != "" {
		awsCredentials = awssdk.Credentials{
			AccessKeyID:     config.AccessKeyID,
			SecretAccessKey: config.SecretAccessKey,
		}
	}

	if config.SessionToken != "" {
		awsCredentials.SessionToken = config.SessionToken
	}

	cfg.Credentials = awssdk.StaticCredentialsProvider{
		Value: awsCredentials,
	}

	awsConfig, err := external.LoadDefaultAWSConfig(cfg)
	if err != nil {
		return nil, err
	}

	awsConfig.Region = config.DefaultRegion

	metricSet := MetricSet{
		BaseMetricSet: base,
		Period:        config.Period,
		AwsConfig:     &awsConfig,
	}

	// Construct MetricSet with a full regions list
	if config.Regions == nil {
		svcEC2 := ec2.New(awsConfig)
		completeRegionsList, err := getRegions(svcEC2)
		if err != nil {
			return nil, err
		}

		metricSet.RegionsList = completeRegionsList
		return &metricSet, nil
	}

	// Construct MetricSet with specific regions list from config
	metricSet.RegionsList = config.Regions
	return &metricSet, nil
}

func getRegions(svc ec2iface.EC2API) (completeRegionsList []string, err error) {
	input := &ec2.DescribeRegionsInput{}
	req := svc.DescribeRegionsRequest(input)
	output, err := req.Send()
	if err != nil {
		err = errors.Wrap(err, "Failed DescribeRegions")
		return
	}
	for _, region := range output.Regions {
		completeRegionsList = append(completeRegionsList, *region.RegionName)
	}
	return
}

// StringInSlice checks if a string is already exists in list
func StringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

// InitEvent initialize mb.Event with basic information like service.name, cloud.provider
func InitEvent(metricsetName string, regionName string) mb.Event {
	event := mb.Event{}
	event.Service = metricsetName
	event.MetricSetFields = common.MapStr{}
	event.RootFields = common.MapStr{}
	event.RootFields.Put("service.name", metricsetName)
	event.RootFields.Put("cloud.provider", "aws")
	if regionName != "" {
		event.RootFields.Put("cloud.region", regionName)
	}
	return event
}
