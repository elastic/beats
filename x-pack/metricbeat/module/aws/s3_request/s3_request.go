// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3_request

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
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
	cfgwarn.Beta("The aws s3_request metricset is beta.")
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
		listMetricsOutputs, err := aws.GetListMetricsOutput(namespace, regionName, svcCloudwatch)
		if err != nil {
			m.logger.Error(err.Error())
			report.Error(err)
			continue
		}

		if listMetricsOutputs == nil || len(listMetricsOutputs) == 0 {
			continue
		}

		metricDataQueries := constructMetricQueries(listMetricsOutputs, m.PeriodInSec)
		// This happens when S3 cloudwatch request metrics are not enabled.
		if len(metricDataQueries) == 0 {
			continue
		}
		// Use metricDataQueries to make GetMetricData API calls
		metricDataOutputs, err := aws.GetMetricDataResults(metricDataQueries, svcCloudwatch, startTime, endTime)
		if err != nil {
			err = errors.Wrap(err, "GetMetricDataResults failed, skipping region "+regionName)
			m.logger.Error(err.Error())
			report.Error(err)
			continue
		}

		// Create Cloudwatch Events for s3_request
		bucketNames := getBucketNames(listMetricsOutputs)
		for _, bucketName := range bucketNames {
			event, err := createS3RequestEvents(metricDataOutputs, regionName, bucketName)
			if err != nil {
				m.logger.Error(err.Error())
				event.Error = err
				report.Event(event)
				continue
			}
			report.Event(event)
		}
	}
}

func getBucketNames(listMetricsOutputs []cloudwatch.Metric) (bucketNames []string) {
	for _, output := range listMetricsOutputs {
		for _, dim := range output.Dimensions {
			if *dim.Name == "BucketName" {
				if aws.StringInSlice(*dim.Value, bucketNames) {
					continue
				}
				bucketNames = append(bucketNames, *dim.Value)
			}
		}
	}
	return
}

func createMetricDataQuery(metric cloudwatch.Metric, periodInSec int, index int) (metricDataQuery cloudwatch.MetricDataQuery) {
	statistic := "Sum"
	period := int64(periodInSec)
	id := "s3r" + strconv.Itoa(index)
	metricDims := metric.Dimensions
	bucketName := ""
	filterID := ""
	for _, dim := range metricDims {
		if *dim.Name == "BucketName" {
			bucketName = *dim.Value
		} else if *dim.Name == "FilterId" {
			filterID = *dim.Value
		}
	}
	metricName := *metric.MetricName
	label := bucketName + " " + filterID + " " + metricName
	metricDataQuery = cloudwatch.MetricDataQuery{
		Id: &id,
		MetricStat: &cloudwatch.MetricStat{
			Period: &period,
			Stat:   &statistic,
			Metric: &metric,
		},
		Label: &label,
	}
	return
}

func constructMetricQueries(listMetricsOutputs []cloudwatch.Metric, periodInSec int) []cloudwatch.MetricDataQuery {
	metricDataQueries := []cloudwatch.MetricDataQuery{}
	metricDataQueryEmpty := cloudwatch.MetricDataQuery{}
	dailyMetricNames := []string{"NumberOfObjects", "BucketSizeBytes"}
	for i, listMetric := range listMetricsOutputs {
		if aws.StringInSlice(*listMetric.MetricName, dailyMetricNames) {
			continue
		}

		metricDataQuery := createMetricDataQuery(listMetric, periodInSec, i)
		if metricDataQuery == metricDataQueryEmpty {
			continue
		}
		metricDataQueries = append(metricDataQueries, metricDataQuery)
	}
	return metricDataQueries
}

// CreateS3Events creates s3_request and s3_daily_storage events from Cloudwatch metric data.
func createS3RequestEvents(outputs []cloudwatch.MetricDataResult, regionName string, bucketName string) (event mb.Event, err error) {
	event.Service = metricsetName
	event.RootFields = common.MapStr{}
	// Cloud fields in ECS
	event.RootFields.Put("service.name", metricsetName)
	event.RootFields.Put("cloud.region", regionName)
	event.RootFields.Put("cloud.provider", "aws")

	// AWS s3_request metrics
	mapOfMetricSetFieldResults := make(map[string]interface{})

	// Find a timestamp for all metrics in output
	timestamp := aws.FindTimestamp(outputs)
	if !timestamp.IsZero() {
		for _, output := range outputs {
			if len(output.Values) == 0 {
				continue
			}
			exists, timestampIdx := aws.CheckTimestampInArray(timestamp, output.Timestamps)
			if exists {
				labels := strings.Split(*output.Label, " ")
				if labels[0] == bucketName && len(output.Values) > timestampIdx {
					mapOfMetricSetFieldResults[labels[2]] = fmt.Sprint(output.Values[timestampIdx])
				}
			}
		}
	}

	resultMetricSetFields, err := aws.EventMapping(mapOfMetricSetFieldResults, schemaMetricSetFields)
	if err != nil {
		err = errors.Wrap(err, "Error trying to apply schema schemaMetricSetFields in AWS s3_request metricbeat module.")
		return
	}

	resultMetricSetFields.Put("bucket.name", bucketName)
	event.MetricSetFields = resultMetricSetFields
	return
}
