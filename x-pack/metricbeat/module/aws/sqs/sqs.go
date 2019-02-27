// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sqs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
)

var metricsetName = "sqs"

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
	logger := logp.NewLogger(aws.ModuleName)
	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}

	// Check if period is set to be multiple of 300s
	remainder := metricSet.PeriodInSec % 300
	if remainder != 0 {
		err := errors.New("period needs to be set to 300s (or a multiple of 300s). " +
			"To avoid data missing or extra costs, please make sure period is set correctly in config.yml")
		logger.Info(err)
	}

	return &MetricSet{
		MetricSet: metricSet,
		logger:    logger,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) {
	namespace := "AWS/SQS"
	// Get startTime and endTime
	startTime, endTime, err := aws.GetStartTimeEndTime(m.DurationString)
	if err != nil {
		m.logger.Error(errors.Wrap(err, "Error ParseDuration"))
		report.Error(err)
		return
	}

	for _, regionName := range m.MetricSet.RegionsList {
		m.MetricSet.AwsConfig.Region = regionName
		svcCloudwatch := cloudwatch.New(*m.MetricSet.AwsConfig)

		// Get listMetrics output
		listMetricsOutput, err := aws.GetListMetricsOutput(namespace, regionName, svcCloudwatch)
		if err != nil {
			m.logger.Error(err.Error())
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
			m.logger.Error(err.Error())
			report.Error(err)
			continue
		}

		// Create Cloudwatch Events for SQS
		event, err := createSQSEvents(metricDataResults, metricsetName, regionName, schemaRequestFields)
		if err != nil {
			m.logger.Error(err.Error())
			event.Error = err
			report.Event(event)
			continue
		}

		report.Event(event)
	}
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
	id := "sqs" + strconv.Itoa(index)
	metricDims := metric.Dimensions
	metricName := *metric.MetricName
	queueName := ""
	for _, dim := range metricDims {
		if *dim.Name == "QueueName" {
			queueName = *dim.Value
		}
	}
	label := queueName + " " + metricName

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

func createSQSEvents(getMetricDataResults []cloudwatch.MetricDataResult, metricsetName string, regionName string, schemaMetricFields s.Schema) (event mb.Event, err error) {
	event.Service = metricsetName
	event.RootFields = common.MapStr{}
	event.RootFields.Put("service.name", metricsetName)
	event.RootFields.Put("cloud.region", regionName)

	mapOfMetricSetFieldResults := make(map[string]interface{})
	for _, output := range getMetricDataResults {
		if len(output.Values) == 0 {
			continue
		}
		labels := strings.Split(*output.Label, " ")
		mapOfMetricSetFieldResults["queue.name"] = labels[0]
		mapOfMetricSetFieldResults[labels[1]] = fmt.Sprint(output.Values[0])
	}

	resultMetricSetFields, err := aws.EventMapping(mapOfMetricSetFieldResults, schemaMetricFields)
	if err != nil {
		err = errors.Wrap(err, "Error trying to apply schemaMetricSetFields in AWS SQS metricbeat module.")
		return
	}
	event.MetricSetFields = resultMetricSetFields
	return
}
