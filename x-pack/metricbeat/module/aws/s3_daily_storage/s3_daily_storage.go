// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3_daily_storage

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/metricbeat/mb"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
)

var metricsetName = "s3_daily_storage"

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(aws.ModuleName, metricsetName, New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	*aws.MetricSet
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	moduleConfig := aws.Config{}
	if err := base.Module().UnpackConfig(&moduleConfig); err != nil {
		return nil, err
	}

	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}

	// Check if period is set to be multiple of 86400s
	remainder := int(metricSet.Period.Seconds()) % 86400
	if remainder != 0 {
		err := errors.New("period needs to be set to 86400s (or a multiple of 86400s). " +
			"To avoid data missing or extra costs, please make sure period is set correctly " +
			"in config.yml")
		base.Logger().Info(err)
	}

	return &MetricSet{
		MetricSet: metricSet,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	namespace := "AWS/S3"
	// Get startTime and endTime
	startTime, endTime := aws.GetStartTimeEndTime(m.Period)

	// GetMetricData for AWS S3 from Cloudwatch
	for _, regionName := range m.MetricSet.RegionsList {
		awsConfig := m.MetricSet.AwsConfig.Copy()
		awsConfig.Region = regionName

		svcCloudwatch := cloudwatch.New(awscommon.EnrichAWSConfigWithEndpoint(
			m.Endpoint, "monitoring", regionName, awsConfig))

		listMetricsOutputs, err := aws.GetListMetricsOutput(namespace, regionName, svcCloudwatch)
		if err != nil {
			err = errors.Wrap(err, "GetListMetricsOutput failed, skipping region "+regionName)
			m.Logger().Error(err.Error())
			report.Error(err)
			continue
		}

		if listMetricsOutputs == nil || len(listMetricsOutputs) == 0 {
			continue
		}

		metricDataQueries := constructMetricQueries(listMetricsOutputs, m.Period)
		// Use metricDataQueries to make GetMetricData API calls
		metricDataOutputs, err := aws.GetMetricDataResults(metricDataQueries, svcCloudwatch, startTime, endTime)
		if err != nil {
			err = errors.Wrap(err, "GetMetricDataResults failed, skipping region "+regionName)
			m.Logger().Error(err)
			report.Error(err)
			continue
		}

		// Create Cloudwatch Events for s3_daily_storage
		bucketNames := getBucketNames(listMetricsOutputs)
		for _, bucketName := range bucketNames {
			event, err := createCloudWatchEvents(metricDataOutputs, regionName, bucketName, m.AccountName, m.AccountID)
			if err != nil {
				err = errors.Wrap(err, "createCloudWatchEvents failed")
				m.Logger().Error(err)
				event.Error = err
				report.Event(event)
				continue
			}

			if reported := report.Event(event); !reported {
				m.Logger().Debug("Fetch interrupted, failed to emit event")
				return nil
			}
		}
	}

	return nil
}

func getBucketNames(listMetricsOutputs []cloudwatch.Metric) (bucketNames []string) {
	for _, output := range listMetricsOutputs {
		for _, dim := range output.Dimensions {
			if *dim.Name == "BucketName" {
				if exists, _ := aws.StringInSlice(*dim.Value, bucketNames); exists {
					continue
				}
				bucketNames = append(bucketNames, *dim.Value)
			}
		}
	}
	return
}

func constructMetricQueries(listMetricsOutputs []cloudwatch.Metric, period time.Duration) []cloudwatch.MetricDataQuery {
	var metricDataQueries []cloudwatch.MetricDataQuery
	metricDataQueryEmpty := cloudwatch.MetricDataQuery{}
	metricNames := []string{"NumberOfObjects", "BucketSizeBytes"}
	for i, listMetric := range listMetricsOutputs {
		if exists, _ := aws.StringInSlice(*listMetric.MetricName, metricNames); !exists {
			continue
		}

		metricDataQuery := createMetricDataQuery(listMetric, period, i)
		if metricDataQuery == metricDataQueryEmpty {
			continue
		}
		metricDataQueries = append(metricDataQueries, metricDataQuery)
	}
	return metricDataQueries
}

func createMetricDataQuery(metric cloudwatch.Metric, period time.Duration, index int) (metricDataQuery cloudwatch.MetricDataQuery) {
	statistic := "Average"
	periodInSec := int64(period.Seconds())
	id := "s3d" + strconv.Itoa(index)
	metricDims := metric.Dimensions
	bucketName := ""
	storageType := ""
	for _, dim := range metricDims {
		if *dim.Name == "BucketName" {
			bucketName = *dim.Value
		} else if *dim.Name == "StorageType" {
			storageType = *dim.Value
		}
	}
	metricName := *metric.MetricName
	label := bucketName + " " + storageType + " " + metricName

	metricDataQuery = cloudwatch.MetricDataQuery{
		Id: &id,
		MetricStat: &cloudwatch.MetricStat{
			Period: &periodInSec,
			Stat:   &statistic,
			Metric: &metric,
		},
		Label: &label,
	}
	return
}

func createCloudWatchEvents(outputs []cloudwatch.MetricDataResult, regionName string, bucketName string, accountName string, accountID string) (event mb.Event, err error) {
	event = aws.InitEvent(regionName, accountName, accountID)

	// AWS s3_daily_storage metrics
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
		err = errors.Wrap(err, "Error trying to apply schema schemaMetricSetFields in AWS s3_daily_storage metricbeat module.")
		return
	}

	event.MetricSetFields = resultMetricSetFields
	event.RootFields.Put("aws.s3.bucket.name", bucketName)
	return
}
