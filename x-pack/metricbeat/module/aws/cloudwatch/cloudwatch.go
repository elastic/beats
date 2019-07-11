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
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
)

var (
	metricsetName        = "cloudwatch"
	metricNameIdx        = 0
	namespaceIdx         = 1
	statisticIdx         = 2
	identifierNameIdx    = 3
	identifierValueIdx   = 4
	defaultStatistics    = []string{"Average", "Maximum", "Minimum", "Sum", "SampleCount"}
	statisticLookupTable = map[string]string{
		"Average":     "avg",
		"Sum":         "sum",
		"Maximum":     "max",
		"Minimum":     "min",
		"SampleCount": "count",
	}
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
	CloudwatchConfigs []Config `config:"metrics" validate:"nonzero,required"`
}

// Dimension holds name and value for cloudwatch metricset dimension config.
type Dimension struct {
	Name  string `config:"name" validate:"nonzero"`
	Value string `config:"value" validate:"nonzero"`
}

// Config holds a configuration specific for cloudwatch metricset.
type Config struct {
	Namespace          string      `config:"namespace" validate:"nonzero,required"`
	MetricName         []string    `config:"name"`
	Dimensions         []Dimension `config:"dimensions"`
	ResourceTypeFilter string      `config:"tags.resource_type_filter"`
	Statistic          []string    `config:"statistic"`
}

type listMetricWithDetail struct {
	cloudwatchMetric   cloudwatch.Metric
	resourceTypeFilter string
	statistic          []string
}

type namespaceWithDetail struct {
	namespace          string
	resourceTypeFilter string
	statistic          []string
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
		CloudwatchMetrics []Config `config:"metrics" validate:"nonzero,required"`
	}{}

	err = base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, errors.Wrap(err, "error unpack raw module config using UnpackConfig")
	}

	if len(config.CloudwatchMetrics) == 0 {
		return nil, errors.New("metrics in config is missing")
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

	// Get listMetricsTotal and namespacesTotal from configuration
	listMetricDetailTotal, namespaceDetailTotal := readCloudwatchConfig(m.CloudwatchConfigs)

	// Create events based on listMetricsTotal from configuration
	for _, regionName := range m.MetricSet.RegionsList {
		awsConfig := m.MetricSet.AwsConfig.Copy()
		awsConfig.Region = regionName
		svcCloudwatch := cloudwatch.New(awsConfig)
		svcResourceAPI := resourcegroupstaggingapi.New(awsConfig)

		err := m.createEvents(svcCloudwatch, svcResourceAPI, listMetricDetailTotal, regionName, startTime, endTime, report)
		if err != nil {
			return errors.Wrap(err, "createEvents failed")
		}
	}

	// Create events based on namespacesTotal from configuration
	for _, namespaceResourceTypeStats := range namespaceDetailTotal {
		for _, regionName := range m.MetricSet.RegionsList {
			awsConfig := m.MetricSet.AwsConfig.Copy()
			awsConfig.Region = regionName
			svcCloudwatch := cloudwatch.New(awsConfig)

			listMetricsOutput, err := aws.GetListMetricsOutput(namespaceResourceTypeStats.namespace, regionName, svcCloudwatch)
			if err != nil {
				m.Logger().Info(err.Error())
				continue
			}

			if listMetricsOutput == nil || len(listMetricsOutput) == 0 {
				continue
			}

			var listMetricDetailTotal []listMetricWithDetail
			for _, listMetric := range listMetricsOutput {
				metricsTypeStats := listMetricWithDetail{
					cloudwatchMetric:   listMetric,
					resourceTypeFilter: namespaceResourceTypeStats.resourceTypeFilter,
					statistic:          namespaceResourceTypeStats.statistic,
				}
				listMetricDetailTotal = append(listMetricDetailTotal, metricsTypeStats)
			}

			svcResourceAPI := resourcegroupstaggingapi.New(awsConfig)
			err = m.createEvents(svcCloudwatch, svcResourceAPI, listMetricDetailTotal, regionName, startTime, endTime, report)
			if err != nil {
				return errors.Wrap(err, "createEvents failed for region "+regionName)
			}
		}
	}

	return nil
}

func readCloudwatchConfig(cloudwatchConfigsTotal []Config) ([]listMetricWithDetail, []namespaceWithDetail) {
	var listMetricDetailTotal []listMetricWithDetail
	var namespaceDetailTotal []namespaceWithDetail

	for _, cloudwatchConfig := range cloudwatchConfigsTotal {
		// If there is no statistic method specified, then use the default.
		if cloudwatchConfig.Statistic == nil {
			cloudwatchConfig.Statistic = defaultStatistics
		}

		if cloudwatchConfig.MetricName != nil {
			listMetricsOutput := convertConfigToListMetrics(cloudwatchConfig, cloudwatchConfig.Namespace)
			for _, listMetric := range listMetricsOutput {
				listMetricDetailTotal = append(listMetricDetailTotal, listMetricWithDetail{
					statistic:          cloudwatchConfig.Statistic,
					resourceTypeFilter: cloudwatchConfig.ResourceTypeFilter,
					cloudwatchMetric:   listMetric,
				})
			}
		} else {
			namespaceDetailTotal = append(namespaceDetailTotal, namespaceWithDetail{
				namespace:          cloudwatchConfig.Namespace,
				resourceTypeFilter: cloudwatchConfig.ResourceTypeFilter,
				statistic:          cloudwatchConfig.Statistic,
			})
		}
	}
	return listMetricDetailTotal, namespaceDetailTotal
}

func constructLabel(metric listMetricWithDetail, statistic string) string {
	// label = metricName + namespace + dimensionKeyTotal + dimensionValueTotal + statistic
	label := *metric.cloudwatchMetric.MetricName + " " + *metric.cloudwatchMetric.Namespace + " " + statistic
	dimNames := ""
	dimValues := ""
	for i, dim := range metric.cloudwatchMetric.Dimensions {
		dimNames += *dim.Name
		dimValues += *dim.Value
		if i != len(metric.cloudwatchMetric.Dimensions)-1 {
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

func createMetricDataQueries(listMetricDetailTotal []listMetricWithDetail, period time.Duration) (metricDataQueries []cloudwatch.MetricDataQuery) {
	for i, listMetricTypeStats := range listMetricDetailTotal {
		for j, statistic := range listMetricTypeStats.statistic {
			stat := new(string)
			*stat = statistic
			label := constructLabel(listMetricTypeStats, statistic)
			periodInSec := int64(period.Seconds())

			id := "cw" + strconv.Itoa(i) + "stats" + strconv.Itoa(j)
			metricDataQueries = append(metricDataQueries, cloudwatch.MetricDataQuery{
				Id: &id,
				MetricStat: &cloudwatch.MetricStat{
					Period: &periodInSec,
					Stat:   stat,
					Metric: &listMetricTypeStats.cloudwatchMetric,
				},
				Label: &label,
			})
		}
	}
	return
}

func getIdentifiers(listMetricDetailTotal []listMetricWithDetail) map[string][]string {
	if len(listMetricDetailTotal) == 0 {
		return nil
	}

	identifiers := map[string][]string{}
	for _, listMetricTypeStats := range listMetricDetailTotal {
		identifierName := ""
		identifierValue := ""
		if len(listMetricTypeStats.cloudwatchMetric.Dimensions) == 0 {
			continue
		}

		for i, dim := range listMetricTypeStats.cloudwatchMetric.Dimensions {
			identifierName += *dim.Name
			identifierValue += *dim.Value
			if i != len(listMetricTypeStats.cloudwatchMetric.Dimensions)-1 {
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

func generateFieldName(labels []string) string {
	stat := labels[statisticIdx]
	// Check if statistic method is one of Sum, SampleCount, Minimum, Maximum, Average
	if statisticMethod, ok := statisticLookupTable[stat]; ok {
		return "metrics." + labels[metricNameIdx] + "." + statisticMethod
	}
	// If not, then it should be a percentile in the form of pN
	return "metrics." + labels[metricNameIdx] + "." + stat
}

func insertMetricSetFields(event mb.Event, metricValue float64, labels []string) mb.Event {
	event.MetricSetFields.Put(generateFieldName(labels), metricValue)
	event.MetricSetFields.Put("namespace", labels[namespaceIdx])
	if len(labels) == 3 {
		return event
	}

	dimNames := strings.Split(labels[identifierNameIdx], ",")
	dimValues := strings.Split(labels[identifierValueIdx], ",")
	for i := 0; i < len(dimNames); i++ {
		event.MetricSetFields.Put("dimensions."+dimNames[i], dimValues[i])
	}
	return event
}

func convertConfigToListMetrics(cloudwatchConfig Config, namespace string) []cloudwatch.Metric {
	// convert config input to []cloudwatch.Metric
	var listMetricsOutputTotal []cloudwatch.Metric
	var cloudwatchDimensions []cloudwatch.Dimension
	for _, dim := range cloudwatchConfig.Dimensions {
		name := dim.Name
		value := dim.Value
		cloudwatchDimensions = append(cloudwatchDimensions, cloudwatch.Dimension{
			Name:  &name,
			Value: &value,
		})
	}

	for _, metricName := range cloudwatchConfig.MetricName {
		listMetricsOutputTotal = append(listMetricsOutputTotal, cloudwatch.Metric{
			Namespace:  &namespace,
			MetricName: &metricName,
			Dimensions: cloudwatchDimensions,
		})
	}
	return listMetricsOutputTotal
}

func (m *MetricSet) createEvents(svcCloudwatch cloudwatchiface.ClientAPI, svcResourceAPI resourcegroupstaggingapiiface.ClientAPI, listMetricDetailTotal []listMetricWithDetail, regionName string, startTime time.Time, endTime time.Time, report mb.ReporterV2) error {
	// Get tags
	var resourceTypes []string
	for _, listMetrics := range listMetricDetailTotal {
		if !aws.StringInSlice(listMetrics.resourceTypeFilter, resourceTypes) {
			resourceTypes = append(resourceTypes, listMetrics.resourceTypeFilter)
		}
	}

	resourceTagMap, err := aws.GetResourcesTags(svcResourceAPI, resourceTypes)
	if err != nil {
		// If GetResourcesTags failed, continue report event just without tags.
		m.Logger().Info(errors.Wrap(err, "getResourcesTags failed, skipping region "+regionName))
	}

	identifiers := getIdentifiers(listMetricDetailTotal)
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
	metricDataQueries := createMetricDataQueries(listMetricDetailTotal, m.Period)
	if len(metricDataQueries) == 0 {
		return nil
	}

	// Use metricDataQueries to make GetMetricData API calls
	metricDataResults, err := aws.GetMetricDataResults(metricDataQueries, svcCloudwatch, startTime, endTime)
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
				if len(labels) == 5 {
					identifierValue := labels[identifierValueIdx]
					events[identifierValue] = insertMetricSetFields(events[identifierValue], output.Values[timestampIdx], labels)
					tags := resourceTagMap[identifierValue]
					for _, tag := range tags {
						events[identifierValue].ModuleFields.Put("tags."+*tag.Key, *tag.Value)
					}
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
