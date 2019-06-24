// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudwatch

import (
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/cloudwatchiface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
)

var (
	metricsetName      = "cloudwatch"
	metricNameIdx      = 0
	namespaceIdx       = 1
	identifierNameIdx  = 2
	identifierValueIdx = 3
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
	CloudwatchConfigs []Config `config:"cloudwatch_metrics" validate:"nonzero,required"`
}

// Dimension holds name and value for cloudwatch metricset dimension config.
type Dimension struct {
	Name  string `config:"name" validate:"nonzero"`
	Value string `config:"value" validate:"nonzero"`
}

// Config holds a configuration specific for cloudwatch metricset.
type Config struct {
	Namespace  string      `config:"namespace" validate:"nonzero,required"`
	MetricName string      `config:"metricname"`
	Dimensions []Dimension `config:"dimensions"`
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The aws cloudwatch metricset is beta.")
	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}

	config := struct {
		CloudwatchMetrics []Config `config:"cloudwatch_metrics" validate:"nonzero,required"`
	}{}

	err = base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, errors.Wrap(err, "error unpack raw module config using UnpackConfig")
	}

	if len(config.CloudwatchMetrics) == 0 {
		return nil, errors.New("cloudwatch_metrics in config is missing")
	}

	return &MetricSet{
		MetricSet:         metricSet,
		CloudwatchConfigs: config.CloudwatchMetrics,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// Get startTime and endTime
	startTime, endTime := aws.GetStartTimeEndTime(m.Period)

	// Get listMetrics and namespacesTotal from configuration
	listMetrics, namespacesTotal := readCloudwatchConfig(m.CloudwatchConfigs)
	for _, regionName := range m.MetricSet.RegionsList {
		awsConfig := m.MetricSet.AwsConfig.Copy()
		awsConfig.Region = regionName
		svcCloudwatch := cloudwatch.New(awsConfig)
		err := createEvents(svcCloudwatch, listMetrics, regionName, m.Period, startTime, endTime, report)
		if err != nil {
			return errors.Wrap(err, "createEvents failed")
		}
	}

	// Use namespaces from config
	for _, namespace := range namespacesTotal {
		for _, regionName := range m.MetricSet.RegionsList {
			awsConfig := m.MetricSet.AwsConfig.Copy()
			awsConfig.Region = regionName
			svcCloudwatch := cloudwatch.New(awsConfig)
			listMetricsOutput, err := aws.GetListMetricsOutput(namespace, regionName, svcCloudwatch)
			if err != nil {
				m.Logger().Info(err.Error())
				continue
			}

			if listMetricsOutput == nil || len(listMetricsOutput) == 0 {
				continue
			}

			err = createEvents(svcCloudwatch, listMetricsOutput, regionName, m.Period, startTime, endTime, report)
			if err != nil {
				return errors.Wrap(err, "createEvents failed for region "+regionName)
			}
		}
	}

	return nil
}

func readCloudwatchConfig(cloudwatchConfigs []Config) ([]cloudwatch.Metric, []string) {
	var listMetrics []cloudwatch.Metric
	var namespacesTotal []string

	for _, cloudwatchConfig := range cloudwatchConfigs {
		namespace := cloudwatchConfig.Namespace
		if cloudwatchConfig.MetricName != "" {
			listMetricsOutput := convertConfigToListMetrics(cloudwatchConfig, namespace)
			listMetrics = append(listMetrics, listMetricsOutput)
		} else {
			namespacesTotal = append(namespacesTotal, namespace)
		}
	}
	return listMetrics, namespacesTotal
}

func constructMetricQueries(listMetricsOutput []cloudwatch.Metric, period time.Duration) []cloudwatch.MetricDataQuery {
	var metricDataQueries []cloudwatch.MetricDataQuery
	for i, listMetric := range listMetricsOutput {
		metricDataQuery := createMetricDataQuery(listMetric, i, period)
		metricDataQueries = append(metricDataQueries, metricDataQuery)
	}
	return metricDataQueries
}

func constructLabel(metric cloudwatch.Metric) string {
	// label = metricName + namespace + dimensionKey1 + dimensionValue1 +
	// dimensionKey2 + dimensionValue2 + ...
	label := *metric.MetricName + " " + *metric.Namespace
	dimNames := ""
	dimValues := ""
	for i, dim := range metric.Dimensions {
		dimNames += *dim.Name
		dimValues += *dim.Value
		if i != len(metric.Dimensions)-1 {
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

func createMetricDataQuery(metric cloudwatch.Metric, index int, period time.Duration) (metricDataQuery cloudwatch.MetricDataQuery) {
	statistic := "Average"
	id := "cw" + strconv.Itoa(index)
	label := constructLabel(metric)
	periodInSec := int64(period.Seconds())

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

func insertMetricSetFields(event mb.Event, metricValue float64, labels []string) mb.Event {
	event.MetricSetFields.Put("metrics."+labels[metricNameIdx], metricValue)
	event.MetricSetFields.Put("namespace", labels[namespaceIdx])
	if len(labels) == 2 {
		return event
	}

	dimNames := strings.Split(labels[identifierNameIdx], ",")
	dimValues := strings.Split(labels[identifierValueIdx], ",")
	for i := 0; i < len(dimNames); i++ {
		event.MetricSetFields.Put("dimensions."+dimNames[i], dimValues[i])
	}
	return event
}

func convertConfigToListMetrics(cloudwatchConfig Config, namespace string) cloudwatch.Metric {
	// convert config input to []cloudwatch.Metric
	var cloudwatchDimensions []cloudwatch.Dimension
	for _, dim := range cloudwatchConfig.Dimensions {
		name := dim.Name
		value := dim.Value
		cloudwatchDimensions = append(cloudwatchDimensions, cloudwatch.Dimension{
			Name:  &name,
			Value: &value,
		})
	}

	listMetricsOutput := cloudwatch.Metric{
		Namespace:  &namespace,
		MetricName: &cloudwatchConfig.MetricName,
		Dimensions: cloudwatchDimensions,
	}
	return listMetricsOutput
}

func createEvents(svc cloudwatchiface.ClientAPI, listMetricsTotal []cloudwatch.Metric, regionName string, period time.Duration, startTime time.Time, endTime time.Time, report mb.ReporterV2) error {
	identifiers := getIdentifiers(listMetricsTotal)
	// Initialize events map per region, which stores one event per identifierValue
	events := map[string]mb.Event{}
	for _, values := range identifiers {
		for _, v := range values {
			events[v] = aws.InitEvent(metricsetName, regionName)
		}
	}
	// Initialize events for the ones without identifiers.
	var eventsNoIdentifier []mb.Event

	// Construct metricDataQueries
	metricDataQueries := constructMetricQueries(listMetricsTotal, period)
	if len(metricDataQueries) == 0 {
		return nil
	}

	// Use metricDataQueries to make GetMetricData API calls
	metricDataResults, err := aws.GetMetricDataResults(metricDataQueries, svc, startTime, endTime)
	if err != nil {
		return errors.Wrap(err, "GetMetricDataResults failed")
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
				if len(labels) == 4 {
					identifierValue := labels[identifierValueIdx]
					events[identifierValue] = insertMetricSetFields(events[identifierValue], output.Values[timestampIdx], labels)
				} else {
					eventNew := aws.InitEvent(metricsetName, regionName)
					eventNew = insertMetricSetFields(eventNew, output.Values[timestampIdx], labels)
					eventsNoIdentifier = append(eventsNoIdentifier, eventNew)
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

	for _, event := range eventsNoIdentifier {
		if len(event.MetricSetFields) != 0 {
			if reported := report.Event(event); !reported {
				return nil
			}
		}
	}

	return nil
}
