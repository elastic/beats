// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudwatch

import (
	"reflect"
	"sort"
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
	metricsetName      = "cloudwatch"
	metricNameIdx      = 0
	namespaceIdx       = 1
	statisticIdx       = 2
	identifierNameIdx  = 3
	identifierValueIdx = 4
	defaultStatistics  = []string{"Average", "Maximum", "Minimum", "Sum", "SampleCount"}
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

type metricsWithStatistics struct {
	cloudwatchMetric cloudwatch.Metric
	statistic        []string
}

type listMetricWithDetail struct {
	metricsWithStats    []metricsWithStatistics
	resourceTypeFilters []string
}

type namespaceWithDetail struct {
	namespace          string
	metricNames        []string
	dimensions         []cloudwatch.Dimension
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

	// Get listMetricDetailTotal and namespaceDetailTotal from configuration
	listMetricDetailTotal, namespaceDetailTotal := m.readCloudwatchConfig()

	// Create events based on listMetricDetailTotal from configuration
	if len(listMetricDetailTotal.metricsWithStats) != 0 {
		for _, regionName := range m.MetricSet.RegionsList {
			awsConfig := m.MetricSet.AwsConfig.Copy()
			awsConfig.Region = regionName
			svcCloudwatch := cloudwatch.New(awsConfig)
			svcResourceAPI := resourcegroupstaggingapi.New(awsConfig)

			err := m.createEvents(svcCloudwatch, svcResourceAPI, listMetricDetailTotal.metricsWithStats, listMetricDetailTotal.resourceTypeFilters, regionName, startTime, endTime, report)
			if err != nil {
				return errors.Wrap(err, "createEvents failed")
			}
		}
	}

	// Create events based on namespaceDetailTotal from configuration
	for _, namespaceResourceType := range namespaceDetailTotal {
		for _, regionName := range m.MetricSet.RegionsList {
			awsConfig := m.MetricSet.AwsConfig.Copy()
			awsConfig.Region = regionName
			svcCloudwatch := cloudwatch.New(awsConfig)

			listMetricsOutput, err := aws.GetListMetricsOutput(namespaceResourceType.namespace, regionName, svcCloudwatch)
			if err != nil {
				m.Logger().Info(err.Error())
				continue
			}

			if listMetricsOutput == nil || len(listMetricsOutput) == 0 {
				continue
			}

			var filteredMetricWithStatsTotal []metricsWithStatistics
			var resourceTypes []string

			if namespaceResourceType.metricNames != nil && namespaceResourceType.dimensions == nil {
				for _, listMetric := range listMetricsOutput {
					if !aws.StringInSlice(*listMetric.MetricName, namespaceResourceType.metricNames) {
						continue
					}
					filteredMetricWithStatsTotal = append(filteredMetricWithStatsTotal,
						metricsWithStatistics{
							cloudwatchMetric: listMetric,
							statistic:        namespaceResourceType.statistic,
						})
					if namespaceResourceType.resourceTypeFilter != "" && !aws.StringInSlice(namespaceResourceType.resourceTypeFilter, resourceTypes) {
						resourceTypes = append(resourceTypes, namespaceResourceType.resourceTypeFilter)
					}
				}
			} else if namespaceResourceType.metricNames == nil && namespaceResourceType.dimensions != nil {
				for _, listMetric := range listMetricsOutput {
					if !compareAWSDimensions(listMetric.Dimensions, namespaceResourceType.dimensions) {
						continue
					}
					filteredMetricWithStatsTotal = append(filteredMetricWithStatsTotal,
						metricsWithStatistics{
							cloudwatchMetric: listMetric,
							statistic:        namespaceResourceType.statistic,
						})
					if namespaceResourceType.resourceTypeFilter != "" && !aws.StringInSlice(namespaceResourceType.resourceTypeFilter, resourceTypes) {
						resourceTypes = append(resourceTypes, namespaceResourceType.resourceTypeFilter)
					}
				}
			} else {
				for _, listMetric := range listMetricsOutput {
					filteredMetricWithStatsTotal = append(filteredMetricWithStatsTotal,
						metricsWithStatistics{
							cloudwatchMetric: listMetric,
							statistic:        namespaceResourceType.statistic,
						})
				}

				if namespaceResourceType.resourceTypeFilter != "" {
					resourceTypes = []string{namespaceResourceType.resourceTypeFilter}
				}
			}

			svcResourceAPI := resourcegroupstaggingapi.New(awsConfig)
			err = m.createEvents(svcCloudwatch, svcResourceAPI, filteredMetricWithStatsTotal, resourceTypes, regionName, startTime, endTime, report)
			if err != nil {
				return errors.Wrap(err, "createEvents failed for region "+regionName)
			}
		}
	}

	return nil
}

func (m *MetricSet) readCloudwatchConfig() (listMetricWithDetail, []namespaceWithDetail) {
	var listMetricDetailTotal listMetricWithDetail
	var namespaceDetailTotal []namespaceWithDetail
	var metricsWithStatsTotal []metricsWithStatistics
	var resourceTypes []string

	for _, config := range m.CloudwatchConfigs {
		// If there is no statistic method specified, then use the default.
		if config.Statistic == nil {
			config.Statistic = defaultStatistics
		}

		var cloudwatchDimensions []cloudwatch.Dimension
		for _, dim := range config.Dimensions {
			name := dim.Name
			value := dim.Value
			cloudwatchDimensions = append(cloudwatchDimensions, cloudwatch.Dimension{
				Name:  &name,
				Value: &value,
			})
		}

		if config.MetricName != nil && config.Dimensions != nil {
			namespace := config.Namespace
			for _, metricName := range config.MetricName {
				metricsWithStats := metricsWithStatistics{
					cloudwatchMetric: cloudwatch.Metric{
						Namespace:  &namespace,
						MetricName: &metricName,
						Dimensions: cloudwatchDimensions,
					},
					statistic: config.Statistic,
				}
				metricsWithStatsTotal = append(metricsWithStatsTotal, metricsWithStats)
			}

			if config.ResourceTypeFilter != "" && !aws.StringInSlice(config.ResourceTypeFilter, resourceTypes) {
				resourceTypes = append(resourceTypes, config.ResourceTypeFilter)
			}
			continue
		}

		if config.MetricName != nil {
			namespaceDetailTotal = append(namespaceDetailTotal, namespaceWithDetail{
				namespace:          config.Namespace,
				metricNames:        config.MetricName,
				dimensions:         cloudwatchDimensions,
				resourceTypeFilter: config.ResourceTypeFilter,
				statistic:          config.Statistic,
			})
			continue
		}

		namespaceDetailTotal = append(namespaceDetailTotal, namespaceWithDetail{
			namespace:          config.Namespace,
			dimensions:         cloudwatchDimensions,
			resourceTypeFilter: config.ResourceTypeFilter,
			statistic:          config.Statistic,
		})

	}

	listMetricDetailTotal.resourceTypeFilters = resourceTypes
	listMetricDetailTotal.metricsWithStats = metricsWithStatsTotal
	return listMetricDetailTotal, namespaceDetailTotal
}

func createMetricDataQueries(listMetricsTotal []metricsWithStatistics, period time.Duration) []cloudwatch.MetricDataQuery {
	var metricDataQueries []cloudwatch.MetricDataQuery
	for i, listMetric := range listMetricsTotal {
		for j, statistic := range listMetric.statistic {
			stat := statistic
			metric := listMetric.cloudwatchMetric
			label := constructLabel(listMetric.cloudwatchMetric, statistic)
			periodInSec := int64(period.Seconds())

			id := "cw" + strconv.Itoa(i) + "stats" + strconv.Itoa(j)
			metricDataQueries = append(metricDataQueries, cloudwatch.MetricDataQuery{
				Id: &id,
				MetricStat: &cloudwatch.MetricStat{
					Period: &periodInSec,
					Stat:   &stat,
					Metric: &metric,
				},
				Label: &label,
			})
		}
	}
	return metricDataQueries
}

func constructLabel(metric cloudwatch.Metric, statistic string) string {
	// label = metricName + namespace + statistic + dimKeys + dimValues
	label := *metric.MetricName + " " + *metric.Namespace + " " + statistic
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

func getIdentifiers(metricsWithStatsTotal []metricsWithStatistics) map[string][]string {
	if len(metricsWithStatsTotal) == 0 {
		return nil
	}

	identifiers := map[string][]string{}
	for _, metricsWithStats := range metricsWithStatsTotal {
		identifierName := ""
		identifierValue := ""
		if len(metricsWithStats.cloudwatchMetric.Dimensions) == 0 {
			continue
		}

		for i, dim := range metricsWithStats.cloudwatchMetric.Dimensions {
			identifierName += *dim.Name
			identifierValue += *dim.Value
			if i != len(metricsWithStats.cloudwatchMetric.Dimensions)-1 {
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

func statisticLookup(stat string) (string, bool) {
	statisticLookupTable := map[string]string{
		"Average":     "avg",
		"Sum":         "sum",
		"Maximum":     "max",
		"Minimum":     "min",
		"SampleCount": "count",
	}
	statMethod, ok := statisticLookupTable[stat]
	if !ok {
		ok = strings.HasPrefix(stat, "p")
		statMethod = stat
	}
	return statMethod, ok
}

func generateFieldName(labels []string) string {
	stat := labels[statisticIdx]
	// Check if statistic method is one of Sum, SampleCount, Minimum, Maximum, Average
	if statMethod, ok := statisticLookup(stat); ok {
		return "aws.metrics." + labels[metricNameIdx] + "." + statMethod
	}
	// If not, then it should be a percentile in the form of pN
	return "metrics." + labels[metricNameIdx] + "." + stat
}

func insertRootFields(event mb.Event, metricValue float64, labels []string) mb.Event {
	event.RootFields.Put(generateFieldName(labels), metricValue)
	event.RootFields.Put("aws.cloudwatch.namespace", labels[namespaceIdx])
	if len(labels) == 3 {
		return event
	}

	dimNames := strings.Split(labels[identifierNameIdx], ",")
	dimValues := strings.Split(labels[identifierValueIdx], ",")
	for i := 0; i < len(dimNames); i++ {
		event.RootFields.Put("aws.cloudwatch.dimensions."+dimNames[i], dimValues[i])
	}
	return event
}

func (m *MetricSet) createEvents(svcCloudwatch cloudwatchiface.ClientAPI, svcResourceAPI resourcegroupstaggingapiiface.ClientAPI, listMetricWithStatsTotal []metricsWithStatistics, resourceTypes []string, regionName string, startTime time.Time, endTime time.Time, report mb.ReporterV2) error {
	// Get tags
	resourceTagMap, err := aws.GetResourcesTags(svcResourceAPI, resourceTypes)
	if err != nil {
		// If GetResourcesTags failed, continue report event just without tags.
		m.Logger().Info(errors.Wrap(err, "getResourcesTags failed, skipping region "+regionName))
	}

	identifiers := getIdentifiers(listMetricWithStatsTotal)
	// Initialize events map per region, which stores one event per identifierValue
	events := map[string]mb.Event{}
	for _, values := range identifiers {
		for _, v := range values {
			events[v] = aws.InitEvent(regionName)
		}
	}
	// Initialize events for the ones without identifiers.
	var eventsNoIdentifier []mb.Event

	// Construct metricDataQueries
	metricDataQueries := createMetricDataQueries(listMetricWithStatsTotal, m.Period)
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
					events[identifierValue] = insertRootFields(events[identifierValue], output.Values[timestampIdx], labels)
					tags := resourceTagMap[identifierValue]
					for _, tag := range tags {
						events[identifierValue].RootFields.Put("aws.tags."+*tag.Key, *tag.Value)
					}
				} else {
					eventNew := aws.InitEvent(regionName)
					eventNew = insertRootFields(eventNew, output.Values[timestampIdx], labels)
					eventsNoIdentifier = append(eventsNoIdentifier, eventNew)
				}
			}
		}
	}

	for _, event := range events {
		if reported := report.Event(event); !reported {
			return nil
		}
	}

	for _, event := range eventsNoIdentifier {
		if reported := report.Event(event); !reported {
			return nil
		}
	}

	return nil
}

func compareAWSDimensions(dim1 []cloudwatch.Dimension, dim2 []cloudwatch.Dimension) bool {
	if len(dim1) != len(dim2) {
		return false
	}
	var dim1String []string
	var dim2String []string
	for i := range dim1 {
		dim1String = append(dim1String, dim1[i].String())
	}
	for i := range dim2 {
		dim2String = append(dim2String, dim2[i].String())
	}

	sort.Strings(dim1String)
	sort.Strings(dim2String)
	return reflect.DeepEqual(dim1String, dim2String)
}
