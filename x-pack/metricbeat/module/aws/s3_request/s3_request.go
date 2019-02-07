// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3_request

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
)

var metricsetName = "s3_request"

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(aws.ModuleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	*aws.MetricSet
	logger *logp.Logger
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The aws s3 metricset is beta.")
	s3Logger := logp.NewLogger(aws.ModuleName)

	moduleConfig := aws.Config{}
	if err := base.Module().UnpackConfig(&moduleConfig); err != nil {
		return nil, err
	}

	if moduleConfig.Period == "" {
		err := errors.New("period is not set in AWS module config")
		s3Logger.Error(err)
	}

	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}

	// Check if period is set to be multiple of 60s
	remainder := metricSet.PeriodInSec % 60
	if remainder != 0 {
		err := errors.New("period needs to be set to 60s (or a multiple of 60s). " +
			"To avoid data missing or extra costs, please make sure period is set correctly " +
			"in config.yml")
		s3Logger.Info(err)
	}

	return &MetricSet{
		MetricSet: metricSet,
		logger:    s3Logger,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) {
	namespace := "AWS/S3"
	// Get startTime and endTime
	startTime, endTime, err := aws.GetStartTimeEndTime(m.DurationString)
	if err != nil {
		logp.Error(errors.Wrap(err, "Error ParseDuration"))
		m.logger.Error(err.Error())
		report.Error(err)
		return
	}

	// GetMetricData for AWS S3 from Cloudwatch
	for _, regionName := range m.MetricSet.RegionsList {
		m.MetricSet.AwsConfig.Region = regionName
		svcCloudwatch := cloudwatch.New(*m.MetricSet.AwsConfig)
		listMetricsOutput, err := aws.GetListMetricsOutput(namespace, regionName, svcCloudwatch)
		if err != nil {
			m.logger.Error(err.Error())
			report.Error(err)
			continue
		}

		if listMetricsOutput == nil || len(listMetricsOutput) == 0 {
			continue
		}

		dailyStorageMetricNames := []string{"NumberOfObjects", "BucketSizeBytes"}
		metricDataQueries := aws.ConstructMetricQueries(listMetricsOutput, int64(m.PeriodInSec), nil, dailyStorageMetricNames)
		if len(metricDataQueries) == 0 {
			continue
		}

		init := true
		getMetricDataOutput := &cloudwatch.GetMetricDataOutput{NextToken: nil}
		for init || getMetricDataOutput.NextToken != nil {
			init = false
			output, err := aws.GetMetricDataPerRegion(metricDataQueries, getMetricDataOutput.NextToken, svcCloudwatch, startTime, endTime)
			if err != nil {
				err = errors.Wrap(err, "getMetricDataPerRegion failed, skipping region "+regionName)
				m.logger.Error(err.Error())
				report.Error(err)
				continue
			}
			getMetricDataOutput.MetricDataResults = append(getMetricDataOutput.MetricDataResults, output.MetricDataResults...)
		}

		// Create Cloudwatch Events for S3
		event, info, err := aws.CreateS3Events(getMetricDataOutput.MetricDataResults, metricsetName, regionName, schemaRootFields, schemaRequestFields)
		if info != "" {
			m.logger.Info(info)
		}

		if err != nil {
			m.logger.Error(err.Error())
			event.Error = err
			report.Event(event)
			continue
		}
		report.Event(event)
	}
}
