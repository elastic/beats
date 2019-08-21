// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sqs

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/sqsiface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	s "github.com/elastic/beats/libbeat/common/schema"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
)

var metricsetName = "sqs"

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
	cfgwarn.Beta("The aws sqs metricset is beta.")

	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}

	// Check if period is set to be multiple of 300s
	remainder := int(metricSet.Period.Seconds()) % 300
	if remainder != 0 {
		err := errors.New("period needs to be set to 300s (or a multiple of 300s). " +
			"To avoid data missing or extra costs, please make sure period is set correctly in config.yml")
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
	namespace := "AWS/SQS"
	// Get startTime and endTime
	startTime, endTime := aws.GetStartTimeEndTime(m.Period)

	for _, regionName := range m.MetricSet.RegionsList {
		awsConfig := m.MetricSet.AwsConfig.Copy()
		awsConfig.Region = regionName
		svcCloudwatch := cloudwatch.New(awsConfig)
		svcSQS := sqs.New(awsConfig)

		// Get queueUrls for each region
		queueURLs, err := getQueueUrls(svcSQS)
		if err != nil {
			m.Logger().Error(err.Error())
			report.Error(err)
			continue
		}
		if len(queueURLs) == 0 {
			continue
		}

		// Get listMetrics output
		listMetricsOutput, err := aws.GetListMetricsOutput(namespace, regionName, svcCloudwatch)
		if err != nil {
			m.Logger().Error(err.Error())
			report.Error(err)
			continue
		}
		if listMetricsOutput == nil || len(listMetricsOutput) == 0 {
			continue
		}

		// Construct metricDataQueries
		metricDataQueries := constructMetricQueries(listMetricsOutput, m.Period)
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

		// Create Cloudwatch Events for SQS
		err = createSQSEvents(queueURLs, metricDataResults, regionName, report)
		if err != nil {
			m.Logger().Debug("Error trying to emit event")
			return nil
		}
	}

	return nil
}

func getQueueUrls(svc sqsiface.ClientAPI) ([]string, error) {
	// ListQueues
	listQueuesInput := &sqs.ListQueuesInput{}
	req := svc.ListQueuesRequest(listQueuesInput)
	output, err := req.Send(context.TODO())
	if err != nil {
		err = errors.Wrap(err, "Error DescribeInstances")
		return nil, err
	}
	return output.QueueUrls, nil
}

func constructMetricQueries(listMetricsOutput []cloudwatch.Metric, period time.Duration) []cloudwatch.MetricDataQuery {
	var metricDataQueries []cloudwatch.MetricDataQuery
	for i, listMetric := range listMetricsOutput {
		metricDataQuery := createMetricDataQuery(listMetric, i, period)
		metricDataQueries = append(metricDataQueries, metricDataQuery)
	}
	return metricDataQueries
}

func createMetricDataQuery(metric cloudwatch.Metric, index int, period time.Duration) (metricDataQuery cloudwatch.MetricDataQuery) {
	statistic := "Sum"
	periodInSec := int64(period.Seconds())
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
			Period: &periodInSec,
			Stat:   &statistic,
			Metric: &metric,
		},
		Label: &label,
	}
	return
}

func createEventPerQueue(getMetricDataResults []cloudwatch.MetricDataResult, queueName string, metricsetName string, regionName string, schemaMetricFields s.Schema) (event mb.Event, err error) {
	event = aws.InitEvent(regionName)

	// AWS sqs metrics
	mapOfMetricSetFieldResults := make(map[string]interface{})

	// Find a timestamp for all metrics in output
	timestamp := aws.FindTimestamp(getMetricDataResults)
	if !timestamp.IsZero() {
		for _, output := range getMetricDataResults {
			if len(output.Values) == 0 {
				continue
			}
			exists, timestampIdx := aws.CheckTimestampInArray(timestamp, output.Timestamps)
			if exists {
				labels := strings.Split(*output.Label, " ")
				if labels[0] == queueName && len(output.Values) > timestampIdx {
					mapOfMetricSetFieldResults[labels[1]] = fmt.Sprint(output.Values[timestampIdx])
				}
			}
		}
	}

	resultMetricSetFields, err := aws.EventMapping(mapOfMetricSetFieldResults, schemaMetricFields)
	if err != nil {
		err = errors.Wrap(err, "Error trying to apply schemaMetricSetFields in AWS SQS metricbeat module.")
		return
	}

	event.MetricSetFields = resultMetricSetFields
	event.MetricSetFields.Put("queue.name", queueName)
	return
}

func createSQSEvents(queueURLs []string, metricDataResults []cloudwatch.MetricDataResult, regionName string, report mb.ReporterV2) error {
	for _, queueURL := range queueURLs {
		queueURLParsed := strings.Split(queueURL, "/")
		queueName := queueURLParsed[len(queueURLParsed)-1]
		event, err := createEventPerQueue(metricDataResults, queueName, metricsetName, regionName, schemaRequestFields)
		if err != nil {
			event.Error = err
			report.Event(event)
			continue
		}

		if reported := report.Event(event); !reported {
			return errors.Wrap(err, "Fetch interrupted, failed to emit event")
		}
	}

	return nil
}
