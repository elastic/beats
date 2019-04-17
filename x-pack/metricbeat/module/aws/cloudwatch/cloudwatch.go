// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudwatch

import (
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
)

var (
	metricsetName = "cloudwatch"
)

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
	cfgwarn.Beta("The aws cloudwatch metricset is beta.")
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
			m.MetricSet.AwsConfig.Region = regionName
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

			// get identifiers from listMetrics
			identifiers := getIdentifiers(listMetricsOutput)

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

			// Initialize events map per region, which stores one event per identifierValue
			events := map[string]mb.Event{}
			for _, values := range identifiers {
				for _, v := range values {
					events[v] = initEvent(regionName)
				}
			}

			// Find a timestamp for all metrics in output
			timestamp := aws.FindTimestamp(metricDataResults)
			if !timestamp.IsZero() {
				for _, output := range metricDataResults {
					if len(output.Values) == 0 {
						continue
					}

					exists, timestampIdx := aws.CheckTimestampInArray(timestamp, output.Timestamps)
					if exists {
						labels := strings.Split(*output.Label, " ")
						if len(labels) == 3 {
							identifierValue := labels[2]
							events[identifierValue] = insertMetricSetFields(events[identifierValue], namespace, output.Values[timestampIdx], labels)
						} else {
							eventNew := initEvent(regionName)
							eventNew = insertMetricSetFields(eventNew, namespace, output.Values[timestampIdx], labels)
							if reported := report.Event(eventNew); !reported {
								return nil
							}
						}
					}
				}
			}

			for _, event := range events {
				if len(event.MetricSetFields) != 0 {
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

func constructLabel(metric cloudwatch.Metric) string {
	metricDims := metric.Dimensions
	metricName := *metric.MetricName
	label := metricName
	dimNames := ""
	dimValues := ""
	for i, dim := range metricDims {
		dimNames += *dim.Name
		dimValues += *dim.Value
		if i != len(metricDims)-1 {
			dimNames += ","
			dimValues += ","
		}
	}

	if dimNames != "" && dimValues != "" {
		label += " " + dimNames
		label += " " + dimValues
	}
	return label
}

func createMetricDataQuery(metric cloudwatch.Metric, index int, period int64) (metricDataQuery cloudwatch.MetricDataQuery) {
	statistic := "Average"
	id := "cw" + strconv.Itoa(index)
	label := constructLabel(metric)

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

func getIdentifiers(listMetricsOutputs []cloudwatch.Metric) map[string][]string {
	if len(listMetricsOutputs) == 0 {
		return nil
	}

	identifiers := map[string][]string{}
	for _, listMetrics := range listMetricsOutputs {
		identifierName := ""
		identifierValue := ""
		if len(listMetrics.Dimensions) == 0 {
			continue
		}

		for i, dim := range listMetrics.Dimensions {
			identifierName += *dim.Name
			identifierValue += *dim.Value
			if i != len(listMetrics.Dimensions)-1 {
				identifierName += ","
				identifierValue += ","
			}

		}

		if identifiers[identifierName] != nil {
			if !aws.StringInSlice(identifierValue, identifiers[identifierName]) {
				identifiers[identifierName] = append(identifiers[identifierName], identifierValue)
			}
		} else {
			identifiers[identifierName] = []string{identifierValue}
		}
	}

	return identifiers
}

func initEvent(regionName string) mb.Event {
	event := mb.Event{}
	event.Service = metricsetName
	event.RootFields = common.MapStr{}
	event.MetricSetFields = common.MapStr{}
	event.RootFields.Put("service.name", metricsetName)
	event.RootFields.Put("cloud.region", regionName)
	return event
}

func insertMetricSetFields(event mb.Event, namespace string, metricValue float64, labels []string) mb.Event {
	event.MetricSetFields.Put("namespace", namespace)
	event.MetricSetFields.Put(labels[0], metricValue)
	if len(labels) == 1 {
		return event
	}

	dimNames := strings.Split(labels[1], ",")
	dimValues := strings.Split(labels[2], ",")
	for i := 0; i < len(dimNames); i++ {
		event.MetricSetFields.Put(dimNames[i], dimValues[i])
	}
	return event
}
