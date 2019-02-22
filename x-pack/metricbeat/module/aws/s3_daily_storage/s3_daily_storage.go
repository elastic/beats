// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3_daily_storage

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
)

var metricsetName = "s3_daily_storage"

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
	cfgwarn.Beta("The aws s3_daily_storage metricset is beta.")
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

	// Check if period is set to be multiple of 86400s
	remainder := metricSet.PeriodInSec % 86400
	if remainder != 0 {
		err := errors.New("period needs to be set to 86400s (or a multiple of 86400s). " +
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
		metricDataQueries := constructMetricQueries(listMetricsOutput, int64(m.PeriodInSec), dailyStorageMetricNames, nil)

		// Use metricDataQueries to make GetMetricData API calls
		metricDataOutput, err := aws.GetMetricDataResults(metricDataQueries, svcCloudwatch, startTime, endTime)
		if err != nil {
			err = errors.Wrap(err, "GetMetricDataResults failed, skipping region "+regionName+" for instance "+instanceID)
			m.logger.Error(err.Error())
			report.Error(err)
			continue
		}

		// Create Cloudwatch Events for S3
		event, info, err := createS3Events(metricDataOutput, metricsetName, regionName, schemaRootFields, schemaMetricSetFields)
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

func constructMetricQueries(listMetricsOutput []cloudwatch.Metric, instanceID string, periodInSec int) []cloudwatch.MetricDataQuery {
	metricDataQueries := []cloudwatch.MetricDataQuery{}
	metricDataQueryEmpty := cloudwatch.MetricDataQuery{}
	for i, listMetric := range listMetricsOutput {
		metricDataQuery := createMetricDataQuery(listMetric, instanceID, i, periodInSec)
		if metricDataQuery == metricDataQueryEmpty {
			continue
		}
		metricDataQueries = append(metricDataQueries, metricDataQuery)
	}
	return metricDataQueries
}
