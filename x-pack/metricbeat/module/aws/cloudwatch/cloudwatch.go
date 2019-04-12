// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudwatch

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

var metricsetName = "cloudwatch"

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
	Namespace string
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}


	return &MetricSet{
		MetricSet: metricSet,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// Get CloudwatchMetrics Config
	cloudwatchConfig := m.CloudwatchMetrics
	if len(cloudwatchConfig) == 0 {
		return errors.New("cloudwatch_metrics in config is missing")
	}
	// Get startTime and endTime
	startTime, endTime, err := aws.GetStartTimeEndTime(m.DurationString)
	if err != nil {
		return errors.Wrap(err, "Error ParseDuration")
	}

	for _, cw := range cloudwatchConfig {
		namespace := cw["namespace"].(string)
		for _, regionName := range m.MetricSet.RegionsList {
			svcCloudwatch := cloudwatch.New(*m.MetricSet.AwsConfig)
			listMetricsOutput, err := aws.GetListMetricsOutput(namespace, regionName, svcCloudwatch)
			if err != nil {
				m.Logger().Errorf(err.Error())
				report.Error(err)
				continue
			}

			if listMetricsOutput == nil || len(listMetricsOutput) == 0 {
				continue
			}

			// Construct metricDataQueries
			metricDataQueries := constructMetricQueries(listMetricsOutput, int64(m.PeriodInSec))
			if len(metricDataQueries) == 0 {
				continue
			}

			// Use metricDataQueries to make GetMetricData API calls
			metricDataResults, err := aws.GetMetricDataResults(metricDataQueries, svcCloudwatch, startTime, endTime)
			if err != nil {
				err = errors.Wrap(err, "GetMetricDataResults failed, skipping region "+regionName)
				m.Logger().Error(err.Error())
				report.Error(err)
				continue
			}

			// Find a timestamp for all metrics in output
			timestamp := aws.FindTimestamp(metricDataResults)
			if !timestamp.IsZero() {
				for _, output := range metricDataResults {
					event := mb.Event{}
					event.Service = metricsetName
					event.RootFields = common.MapStr{}
					event.RootFields.Put("service.name", metricsetName)
					event.RootFields.Put("cloud.region", regionName)

					if len(output.Values) == 0 {
						continue
					}

					exists, timestampIdx := aws.CheckTimestampInArray(timestamp, output.Timestamps)
					if exists {
						event.MetricSetFields = common.MapStr{}
						event.MetricSetFields.Put("namespace", namespace)
						labels := strings.Split(*output.Label, " ")
						for i := 0; i < len(labels)/2; i++ {
							event.MetricSetFields.Put(labels[i+1], labels[i+2])
						}
						if len(output.Values) > timestampIdx {
							event.MetricSetFields.Put(labels[0], output.Values[timestampIdx])
						}
					}

					if reported := report.Event(event); !reported {
						return nil
					}
				}
			}
		}
	}

	return nil
}

func constructMetricQueries(listMetricsOutput []cloudwatch.Metric, period int64) []cloudwatch.MetricDataQuery {
	metricDataQueries := []cloudwatch.MetricDataQuery{}
	for i, listMetric := range listMetricsOutput {
		metricDataQuery := createMetricDataQuery(listMetric, i, period)
		metricDataQueries = append(metricDataQueries, metricDataQuery)
	}
	return metricDataQueries
}

func createMetricDataQuery(metric cloudwatch.Metric, index int, period int64) (metricDataQuery cloudwatch.MetricDataQuery) {
	statistic := "Average"
	id := "cw" + strconv.Itoa(index)
	metricDims := metric.Dimensions
	metricName := *metric.MetricName
	label := metricName + " "
	for _, dim := range metricDims {
		label += *dim.Name
		label += " " + *dim.Value
	}

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
