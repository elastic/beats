// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"strconv"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/metricbeat/mb"
)

// Config defines all required and optional parameters for aws metricsets
type Config struct {
	Period          string `config:"period"`
	AccessKeyID     string `config:"access_key_id"`
	SecretAccessKey string `config:"secret_access_key"`
	SessionToken    string `config:"session_token"`
	DefaultRegion   string `config:"default_region"`
}

// MetricSet is the base metricset for all aws metricsets
type MetricSet struct {
	mb.BaseMetricSet
	RegionsList    []string
	DurationString string
	PeriodInSec    int
	AwsConfig      *awssdk.Config
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

	awsConfig := defaults.Config()
	awsCreds := awssdk.Credentials{
		AccessKeyID:     config.AccessKeyID,
		SecretAccessKey: config.SecretAccessKey,
	}
	if config.SessionToken != "" {
		awsCreds.SessionToken = config.SessionToken
	}

	awsConfig.Credentials = awssdk.StaticCredentialsProvider{
		Value: awsCreds,
	}

	awsConfig.Region = config.DefaultRegion

	svcEC2 := ec2.New(awsConfig)
	regionsList, err := getRegions(svcEC2)
	if err != nil {
		return nil, err
	}

	// Calculate duration based on period
	if config.Period == "" {
		err = errors.New("period is not set in AWS module config")
		return nil, err
	}

	durationString, periodSec, err := convertPeriodToDuration(config.Period)
	if err != nil {
		return nil, err
	}

	// Construct MetricSet
	metricSet := MetricSet{
		BaseMetricSet:  base,
		RegionsList:    regionsList,
		DurationString: durationString,
		PeriodInSec:    periodSec,
		AwsConfig:      &awsConfig,
	}
	return &metricSet, nil
}

func getRegions(svc ec2iface.EC2API) (regionsList []string, err error) {
	input := &ec2.DescribeRegionsInput{}
	req := svc.DescribeRegionsRequest(input)
	output, err := req.Send()
	if err != nil {
		err = errors.Wrap(err, "Failed DescribeRegions")
		return
	}
	for _, region := range output.Regions {
		regionsList = append(regionsList, *region.RegionName)
	}
	return
}

func convertPeriodToDuration(period string) (string, int, error) {
	// Set starttime double the default frequency earlier than the endtime in order to make sure
	// GetMetricDataRequest gets the latest data point for each metric.
	numberPeriod, err := strconv.Atoi(period[0 : len(period)-1])
	if err != nil {
		return "", 0, err
	}

	unitPeriod := period[len(period)-1:]
	switch unitPeriod {
	case "s":
		duration := "-" + strconv.Itoa(numberPeriod*2) + unitPeriod
		return duration, numberPeriod, nil
	case "m":
		duration := "-" + strconv.Itoa(numberPeriod*2) + unitPeriod
		periodInSec := numberPeriod * 60
		return duration, periodInSec, nil
	default:
		err = errors.New("invalid period in config. Please reset period in config")
		duration := "-" + strconv.Itoa(numberPeriod*2) + "s"
		return duration, numberPeriod, err
	}
}
