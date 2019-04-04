// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package rds

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
)

var metricsetName = "rds"

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
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}

	// Check if period is set to be multiple of 60s
	remainder := metricSet.PeriodInSec % 60
	if remainder != 0 {
		err := errors.New("Period needs to be set to 60s (or a multiple of 60s). To avoid data missing or " +
			"extra costs, please make sure period is set correctly in config.yml")
		base.Logger().Info(err)
	}

	return &MetricSet{
		MetricSet: metricSet,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) {
	// Get startTime and endTime
	startTime, endTime, err := aws.GetStartTimeEndTime(m.DurationString)
	if err != nil {
		m.Logger().Error(errors.Wrap(err, "Error ParseDuration"))
		report.Error(err)
		return
	}

	for _, regionName := range m.MetricSet.RegionsList {
		m.MetricSet.AwsConfig.Region = regionName
		svc := rds.New(*m.MetricSet.AwsConfig)

		describeInstanceInput := &rds.DescribeDBInstancesInput{}
		req := svc.DescribeDBInstancesRequest(describeInstanceInput)
		output, err := req.Send()
		if err != nil {
			err = errors.Wrap(err, "DescribeDBInstancesRequest failed, skipping region "+regionName)
			m.Logger().Errorf(err.Error())
			report.Error(err)
			continue
		}
		if len(output.DBInstances) == 0 {
			continue
		}
		// get DBInstance ARN
		dbInstanceArns := []string{}
		for _, dbInstance := range output.DBInstances {
			dbInstanceArns = append(dbInstanceArns, *dbInstance.DBInstanceArn)
		}

		svcCloudwatch := cloudwatch.New(*m.MetricSet.AwsConfig)
		namespace := "AWS/RDS"
		listMetricsOutput, err := aws.GetListMetricsOutput(namespace, regionName, svcCloudwatch)
		if err != nil {
			m.Logger().Error(err.Error())
			report.Error(err)
			continue
		}

		if listMetricsOutput == nil || len(listMetricsOutput) == 0 {
			continue
		}

		for _, dbInstanceArn := range dbInstanceArns {
			metricDataQueries := constructMetricQueries(listMetricsOutput, dbInstanceArn, m.PeriodInSec)
			// If metricDataQueries, still needs to createCloudWatchEvents.
			metricDataOutput := []cloudwatch.MetricDataResult{}
			if len(metricDataQueries) != 0 {
				// Use metricDataQueries to make GetMetricData API calls
				metricDataOutput, err = aws.GetMetricDataResults(metricDataQueries, svcCloudwatch, startTime, endTime)
				if err != nil {
					err = errors.Wrap(err, "GetMetricDataResults failed, skipping region "+regionName)
					m.Logger().Error(err.Error())
					report.Error(err)
					continue
				}
			}
			// Create Cloudwatch Events for RDS
			event, err := createCloudWatchEvents(metricDataOutput, dbInstanceArn, regionName)
			if err != nil {
				m.Logger().Error(err.Error())
				report.Error(err)
				continue
			}
			report.Event(event)
		}
	}
}

func constructMetricQueries(listMetricsOutput []cloudwatch.Metric, dbInstanceArn string, periodInSec int) []cloudwatch.MetricDataQuery {
	metricDataQueries := []cloudwatch.MetricDataQuery{}
	metricDataQueryEmpty := cloudwatch.MetricDataQuery{}
	for i, listMetric := range listMetricsOutput {
		metricDataQuery := createMetricDataQuery(listMetric, i, dbInstanceArn, periodInSec)
		if metricDataQuery == metricDataQueryEmpty {
			continue
		}
		metricDataQueries = append(metricDataQueries, metricDataQuery)
	}
	return metricDataQueries
}

func createMetricDataQuery(metric cloudwatch.Metric, index int, dbInstanceArn string, periodInSec int) cloudwatch.MetricDataQuery {
	statistic := "Average"
	period := int64(periodInSec)
	id := "rds" + strconv.Itoa(index)
	metricDims := metric.Dimensions

	metricDataQuery := cloudwatch.MetricDataQuery{
		Id: &id,
		MetricStat: &cloudwatch.MetricStat{
			Period: &period,
			Stat:   &statistic,
			Metric: &metric,
		},
	}

	label := constructLabel(metricDims, dbInstanceArn, *metric.MetricName)
	metricDataQuery.Label = &label
	return metricDataQuery
}

func constructLabel(metricDimensions []cloudwatch.Dimension, dbInstanceArn string, metricName string) string {
	label := dbInstanceArn + " " + metricName
	if len(metricDimensions) != 0 {
		for _, dim := range metricDimensions {
			label += " "
			label += *dim.Name + " " + *dim.Value
		}
	}
	return label
}

func createCloudWatchEvents(getMetricDataResults []cloudwatch.MetricDataResult, dbInstanceArn string, regionName string) (mb.Event, error) {
	event := mb.Event{}
	event.Service = metricsetName
	event.RootFields = common.MapStr{}

	event.RootFields.Put("service.name", metricsetName)
	event.RootFields.Put("cloud.provider", "aws")
	event.RootFields.Put("cloud.region", regionName)
	event.RootFields.Put("db_instance_arn", dbInstanceArn)

	// AWS RDS Metrics
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
				if len(output.Values) > timestampIdx && len(labels) > 1 {
					mapOfMetricSetFieldResults[labels[1]] = fmt.Sprint(output.Values[timestampIdx])
					for i := 1; i <= (len(labels)-2)/2; i++ {
						mapOfMetricSetFieldResults[labels[i*2]] = labels[(i*2 + 1)]
					}
				}
			}
		}
	}

	resultMetricSetFields, err := aws.EventMapping(mapOfMetricSetFieldResults, schemaMetricSetFields)
	if err != nil {
		err = errors.Wrap(err, "Error trying to apply schema schemaMetricSetFields in AWS EC2 metricbeat module.")
		return event, err
	}

	event.MetricSetFields = resultMetricSetFields
	return event, err
}
