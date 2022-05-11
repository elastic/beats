// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudwatch

import (
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	metricsetName          = "cloudwatch"
	metricNameIdx          = 0
	namespaceIdx           = 1
	statisticIdx           = 2
	identifierNameIdx      = 3
	identifierValueIdx     = 4
	defaultStatistics      = []string{"Average", "Maximum", "Minimum", "Sum", "SampleCount"}
	labelSeparator         = "|"
	dimensionSeparator     = ","
	dimensionValueWildcard = "*"
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
	logger            *logp.Logger
	CloudwatchConfigs []Config `config:"metrics" validate:"nonzero,required"`
}

// Dimension holds name and value for cloudwatch metricset dimension config.
type Dimension struct {
	Name  string `config:"name" validate:"nonzero"`
	Value string `config:"value" validate:"nonzero"`
}

// Config holds a configuration specific for cloudwatch metricset.
type Config struct {
	Namespace    string      `config:"namespace" validate:"nonzero,required"`
	MetricName   []string    `config:"name"`
	Dimensions   []Dimension `config:"dimensions"`
	ResourceType string      `config:"resource_type"`
	Statistic    []string    `config:"statistic"`
}

type metricsWithStatistics struct {
	cloudwatchMetric cloudwatch.Metric
	statistic        []string
	tags             []aws.Tag
}

type listMetricWithDetail struct {
	metricsWithStats    []metricsWithStatistics
	resourceTypeFilters map[string][]aws.Tag
}

// namespaceDetail collects configuration details for each namespace
type namespaceDetail struct {
	resourceTypeFilter string
	names              []string
	tags               []aws.Tag
	statistics         []string
	dimensions         []cloudwatch.Dimension
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logger := logp.NewLogger(metricsetName)
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

	logger.Debugf("cloudwatch config = %s", config)
	if len(config.CloudwatchMetrics) == 0 {
		return nil, errors.New("metrics in config is missing")
	}

	return &MetricSet{
		MetricSet:         metricSet,
		logger:            logger,
		CloudwatchConfigs: config.CloudwatchMetrics,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// Get startTime and endTime
	startTime, endTime := aws.GetStartTimeEndTime(m.Period, m.Latency)
	m.Logger().Debugf("startTime = %s, endTime = %s", startTime, endTime)

	// Check statistic method in config
	err := m.checkStatistics()
	if err != nil {
		return errors.Wrap(err, "checkStatistics failed")
	}

	// Get listMetricDetailTotal and namespaceDetailTotal from configuration
	listMetricDetailTotal, namespaceDetailTotal := m.readCloudwatchConfig()
	m.logger.Debugf("listMetricDetailTotal = %s", listMetricDetailTotal)
	m.logger.Debugf("namespaceDetailTotal = %s", namespaceDetailTotal)

	var config aws.Config
	err = m.Module().UnpackConfig(&config)
	if err != nil {
		return err
	}

	// Create events based on listMetricDetailTotal from configuration
	if len(listMetricDetailTotal.metricsWithStats) != 0 {
		for _, regionName := range m.MetricSet.RegionsList {
			m.logger.Debugf("Collecting metrics from AWS region %s", regionName)
			awsConfig := m.MetricSet.AwsConfig.Copy()
			awsConfig.Region = regionName
			monitoringServiceName := awscommon.CreateServiceName("monitoring", config.AWSConfig.FIPSEnabled, regionName)

			svcCloudwatch := cloudwatch.New(awscommon.EnrichAWSConfigWithEndpoint(
				m.Endpoint, monitoringServiceName, regionName, awsConfig))

			svcResourceAPI := resourcegroupstaggingapi.New(awscommon.EnrichAWSConfigWithEndpoint(
				m.Endpoint, "tagging", regionName, awsConfig)) //Does not support FIPS

			eventsWithIdentifier, err := m.createEvents(svcCloudwatch, svcResourceAPI, listMetricDetailTotal.metricsWithStats, listMetricDetailTotal.resourceTypeFilters, regionName, startTime, endTime)
			if err != nil {
				return errors.Wrap(err, "createEvents failed for region "+regionName)
			}

			m.logger.Debugf("Collected metrics of metrics = %d", len(eventsWithIdentifier))

			err = reportEvents(eventsWithIdentifier, report)
			if err != nil {
				return errors.Wrap(err, "reportEvents failed")
			}
		}
	}

	// Create events based on namespaceDetailTotal from configuration
	for _, regionName := range m.MetricSet.RegionsList {
		m.logger.Debugf("Collecting metrics from AWS region %s", regionName)
		awsConfig := m.MetricSet.AwsConfig.Copy()
		awsConfig.Region = regionName

		monitoringServiceName := awscommon.CreateServiceName("monitoring", config.AWSConfig.FIPSEnabled, regionName)
		svcCloudwatch := cloudwatch.New(awscommon.EnrichAWSConfigWithEndpoint(
			m.Endpoint, monitoringServiceName, regionName, awsConfig))

		svcResourceAPI := resourcegroupstaggingapi.New(awscommon.EnrichAWSConfigWithEndpoint(
			m.Endpoint, "tagging", regionName, awsConfig)) //Does not support FIPS

		for namespace, namespaceDetails := range namespaceDetailTotal {
			m.logger.Debugf("Collected metrics from namespace %s", namespace)

			listMetricsOutput, err := aws.GetListMetricsOutput(namespace, regionName, svcCloudwatch)
			if err != nil {
				m.logger.Info(err.Error())
				continue
			}

			if listMetricsOutput == nil || len(listMetricsOutput) == 0 {
				continue
			}

			// filter listMetricsOutput by detailed configuration per each namespace
			filteredMetricWithStatsTotal := filterListMetricsOutput(listMetricsOutput, namespaceDetails)
			// get resource type filters and tags filters for each namespace
			resourceTypeTagFilters := constructTagsFilters(namespaceDetails)

			eventsWithIdentifier, err := m.createEvents(svcCloudwatch, svcResourceAPI, filteredMetricWithStatsTotal, resourceTypeTagFilters, regionName, startTime, endTime)
			if err != nil {
				return errors.Wrap(err, "createEvents failed for region "+regionName)
			}

			m.logger.Debugf("Collected number of metrics = %d", len(eventsWithIdentifier))

			err = reportEvents(addMetadata(namespace, m.Endpoint, regionName, awsConfig, config.AWSConfig.FIPSEnabled, eventsWithIdentifier), report)
			if err != nil {
				return errors.Wrap(err, "reportEvents failed")
			}
		}
	}
	return nil
}

// filterListMetricsOutput compares config details with listMetricsOutput and filter out the ones don't match
func filterListMetricsOutput(listMetricsOutput []cloudwatch.Metric, namespaceDetails []namespaceDetail) []metricsWithStatistics {
	var filteredMetricWithStatsTotal []metricsWithStatistics
	for _, listMetric := range listMetricsOutput {
		for _, configPerNamespace := range namespaceDetails {
			if configPerNamespace.names != nil && configPerNamespace.dimensions == nil {
				// if metric names are given in config but no dimensions, filter
				// out the metrics with other names
				if exists, _ := aws.StringInSlice(*listMetric.MetricName, configPerNamespace.names); !exists {
					continue
				}
				filteredMetricWithStatsTotal = append(filteredMetricWithStatsTotal,
					metricsWithStatistics{
						cloudwatchMetric: listMetric,
						statistic:        configPerNamespace.statistics,
						tags:             configPerNamespace.tags,
					})

			} else if configPerNamespace.names == nil && configPerNamespace.dimensions != nil {
				// if metric names are not given in config but dimensions are
				// given, only keep the metrics with matching dimensions
				if !compareAWSDimensions(listMetric.Dimensions, configPerNamespace.dimensions) {
					continue
				}
				filteredMetricWithStatsTotal = append(filteredMetricWithStatsTotal,
					metricsWithStatistics{
						cloudwatchMetric: listMetric,
						statistic:        configPerNamespace.statistics,
						tags:             configPerNamespace.tags,
					})
			} else if configPerNamespace.names != nil && configPerNamespace.dimensions != nil {
				if exists, _ := aws.StringInSlice(*listMetric.MetricName, configPerNamespace.names); !exists {
					continue
				}
				if !compareAWSDimensions(listMetric.Dimensions, configPerNamespace.dimensions) {
					continue
				}
				filteredMetricWithStatsTotal = append(filteredMetricWithStatsTotal,
					metricsWithStatistics{
						cloudwatchMetric: listMetric,
						statistic:        configPerNamespace.statistics,
						tags:             configPerNamespace.tags,
					})
			} else {
				// if no metric name and no dimensions given, then keep all listMetricsOutput
				filteredMetricWithStatsTotal = append(filteredMetricWithStatsTotal,
					metricsWithStatistics{
						cloudwatchMetric: listMetric,
						statistic:        configPerNamespace.statistics,
						tags:             configPerNamespace.tags,
					})
			}
		}
	}
	return filteredMetricWithStatsTotal
}

// Collect resource type filters and tag filters from config for cloudwatch
func constructTagsFilters(namespaceDetails []namespaceDetail) map[string][]aws.Tag {
	resourceTypeTagFilters := map[string][]aws.Tag{}
	for _, configPerNamespace := range namespaceDetails {
		if configPerNamespace.resourceTypeFilter != "" {
			if _, ok := resourceTypeTagFilters[configPerNamespace.resourceTypeFilter]; ok {
				resourceTypeTagFilters[configPerNamespace.resourceTypeFilter] = append(resourceTypeTagFilters[configPerNamespace.resourceTypeFilter], configPerNamespace.tags...)
			} else {
				resourceTypeTagFilters[configPerNamespace.resourceTypeFilter] = configPerNamespace.tags
			}
		}
	}
	return resourceTypeTagFilters
}

func (m *MetricSet) checkStatistics() error {
	for _, config := range m.CloudwatchConfigs {
		for _, stat := range config.Statistic {
			if _, ok := statisticLookup(stat); !ok {
				return errors.New("statistic method specified is not valid: " + stat)
			}
		}
	}
	return nil
}

func (m *MetricSet) readCloudwatchConfig() (listMetricWithDetail, map[string][]namespaceDetail) {
	var listMetricDetailTotal listMetricWithDetail
	namespaceDetailTotal := map[string][]namespaceDetail{}
	var metricsWithStatsTotal []metricsWithStatistics
	resourceTypesWithTags := map[string][]aws.Tag{}

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
		// if any Dimension value contains wildcard, then compare dimensions with
		// listMetrics result in filterListMetricsOutput
		if config.MetricName != nil && config.Dimensions != nil &&
			!configDimensionValueContainsWildcard(config.Dimensions) {
			namespace := config.Namespace
			for i := range config.MetricName {
				metricsWithStats := metricsWithStatistics{
					cloudwatchMetric: cloudwatch.Metric{
						Namespace:  &namespace,
						MetricName: &config.MetricName[i],
						Dimensions: cloudwatchDimensions,
					},
					statistic: config.Statistic,
				}
				metricsWithStatsTotal = append(metricsWithStatsTotal, metricsWithStats)
			}

			if config.ResourceType != "" {
				if _, ok := resourceTypesWithTags[config.ResourceType]; ok {
					resourceTypesWithTags[config.ResourceType] = m.MetricSet.TagsFilter
				} else {
					resourceTypesWithTags[config.ResourceType] = append(resourceTypesWithTags[config.ResourceType], m.MetricSet.TagsFilter...)
				}
			}
			continue
		}

		configPerNamespace := namespaceDetail{
			names:              config.MetricName,
			tags:               m.MetricSet.TagsFilter,
			statistics:         config.Statistic,
			resourceTypeFilter: config.ResourceType,
			dimensions:         cloudwatchDimensions,
		}

		if _, ok := namespaceDetailTotal[config.Namespace]; ok {
			namespaceDetailTotal[config.Namespace] = append(namespaceDetailTotal[config.Namespace], configPerNamespace)
		} else {
			namespaceDetailTotal[config.Namespace] = []namespaceDetail{configPerNamespace}
		}
	}

	listMetricDetailTotal.resourceTypeFilters = resourceTypesWithTags
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
	label := *metric.MetricName + labelSeparator + *metric.Namespace + labelSeparator + statistic
	dimNames := ""
	dimValues := ""
	for i, dim := range metric.Dimensions {
		dimNames += *dim.Name
		dimValues += *dim.Value
		if i != len(metric.Dimensions)-1 {
			dimNames += dimensionSeparator
			dimValues += dimensionSeparator
		}
	}

	if dimNames != "" && dimValues != "" {
		label += labelSeparator + dimNames
		label += labelSeparator + dimValues
	}
	return label
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

func generateFieldName(namespace string, labels []string) string {
	stat := labels[statisticIdx]
	// Check if statistic method is one of Sum, SampleCount, Minimum, Maximum, Average
	// With checkStatistics function, no need to check bool return value here
	statMethod, _ := statisticLookup(stat)
	// By default, replace dot "." using underscore "_" for metric names
	return "aws." + stripNamespace(namespace) + ".metrics." + common.DeDot(labels[metricNameIdx]) + "." + statMethod
}

// stripNamespace converts Cloudwatch namespace into the root field we will use for metrics
// example AWS/EC2 -> ec2
func stripNamespace(namespace string) string {
	parts := strings.Split(namespace, "/")
	return strings.ToLower(parts[len(parts)-1])
}

func insertRootFields(event mb.Event, metricValue float64, labels []string) mb.Event {
	namespace := labels[namespaceIdx]
	event.RootFields.Put(generateFieldName(namespace, labels), metricValue)
	event.RootFields.Put("aws.cloudwatch.namespace", namespace)
	if len(labels) == 3 {
		return event
	}

	dimNames := strings.Split(labels[identifierNameIdx], ",")
	dimValues := strings.Split(labels[identifierValueIdx], ",")
	for i := 0; i < len(dimNames); i++ {
		event.RootFields.Put("aws.dimensions."+dimNames[i], dimValues[i])
	}
	return event
}

func (m *MetricSet) createEvents(svcCloudwatch cloudwatchiface.ClientAPI, svcResourceAPI resourcegroupstaggingapiiface.ClientAPI, listMetricWithStatsTotal []metricsWithStatistics, resourceTypeTagFilters map[string][]aws.Tag, regionName string, startTime time.Time, endTime time.Time) (map[string]mb.Event, error) {
	// Initialize events for each identifier.
	events := map[string]mb.Event{}

	// Construct metricDataQueries
	metricDataQueries := createMetricDataQueries(listMetricWithStatsTotal, m.Period)
	m.logger.Debugf("Number of MetricDataQueries = %d", len(metricDataQueries))
	if len(metricDataQueries) == 0 {
		return events, nil
	}

	// Use metricDataQueries to make GetMetricData API calls
	metricDataResults, err := aws.GetMetricDataResults(metricDataQueries, svcCloudwatch, startTime, endTime)
	m.logger.Debugf("Number of metricDataResults = %d", len(metricDataResults))
	if err != nil {
		return events, errors.Wrap(err, "GetMetricDataResults failed")
	}

	// Find a timestamp for all metrics in output
	timestamp := aws.FindTimestamp(metricDataResults)
	if timestamp.IsZero() {
		return nil, nil
	}

	// Create events when there is no tags_filter or resource_type specified.
	if len(resourceTypeTagFilters) == 0 {
		for _, output := range metricDataResults {
			if len(output.Values) == 0 {
				continue
			}

			exists, timestampIdx := aws.CheckTimestampInArray(timestamp, output.Timestamps)
			if exists {
				labels := strings.Split(*output.Label, labelSeparator)
				if len(labels) != 5 {
					// when there is no identifier value in label, use region+accountID+namespace instead
					identifier := regionName + m.AccountID + labels[namespaceIdx]
					if _, ok := events[identifier]; !ok {
						events[identifier] = aws.InitEvent(regionName, m.AccountName, m.AccountID, timestamp)
					}
					events[identifier] = insertRootFields(events[identifier], output.Values[timestampIdx], labels)
					continue
				}

				identifierValue := labels[identifierValueIdx]
				if _, ok := events[identifierValue]; !ok {
					events[identifierValue] = aws.InitEvent(regionName, m.AccountName, m.AccountID, timestamp)
				}
				events[identifierValue] = insertRootFields(events[identifierValue], output.Values[timestampIdx], labels)
			}
		}
		return events, nil
	}

	// Create events with tags
	for resourceType, tagsFilter := range resourceTypeTagFilters {
		m.logger.Debugf("resourceType = %s", resourceType)
		m.logger.Debugf("tagsFilter = %s", tagsFilter)
		resourceTagMap, err := aws.GetResourcesTags(svcResourceAPI, []string{resourceType})
		if err != nil {
			// If GetResourcesTags failed, continue report event just without tags.
			m.logger.Info(errors.Wrap(err, "getResourcesTags failed, skipping region "+regionName))
		}

		if len(tagsFilter) != 0 && len(resourceTagMap) == 0 {
			continue
		}

		// filter resourceTagMap
		for identifier, tags := range resourceTagMap {
			if exists := aws.CheckTagFiltersExist(tagsFilter, tags); !exists {
				m.logger.Debugf("In region %s, service %s tags does not match tags_filter", regionName, identifier)
				delete(resourceTagMap, identifier)
				continue
			}
			m.logger.Debugf("In region %s, service %s tags match tags_filter", regionName, identifier)
		}

		for _, output := range metricDataResults {
			if len(output.Values) == 0 {
				continue
			}

			exists, timestampIdx := aws.CheckTimestampInArray(timestamp, output.Timestamps)
			if exists {
				labels := strings.Split(*output.Label, labelSeparator)
				if len(labels) != 5 {
					// if there is no tag in labels but there is a tagsFilter, then no event should be reported.
					if len(tagsFilter) != 0 {
						continue
					}

					// when there is no identifier value in label, use region+accountID+namespace instead
					identifier := regionName + m.AccountID + labels[namespaceIdx]
					if _, ok := events[identifier]; !ok {
						events[identifier] = aws.InitEvent(regionName, m.AccountName, m.AccountID, timestamp)
					}
					events[identifier] = insertRootFields(events[identifier], output.Values[timestampIdx], labels)
					continue
				}

				identifierValue := labels[identifierValueIdx]
				if _, ok := events[identifierValue]; !ok {
					// when tagsFilter is not empty but no entry in
					// resourceTagMap for this identifier, do not initialize
					// an event for this identifier.
					if len(tagsFilter) != 0 && resourceTagMap[identifierValue] == nil {
						continue
					}
					events[identifierValue] = aws.InitEvent(regionName, m.AccountName, m.AccountID, timestamp)
				}
				events[identifierValue] = insertRootFields(events[identifierValue], output.Values[timestampIdx], labels)

				// add tags to event based on identifierValue
				insertTags(events, identifierValue, resourceTagMap)
			}
		}
	}
	return events, nil
}

func reportEvents(eventsWithIdentifier map[string]mb.Event, report mb.ReporterV2) error {
	for _, event := range eventsWithIdentifier {
		if reported := report.Event(event); !reported {
			return nil
		}
	}
	return nil
}

func configDimensionValueContainsWildcard(dim []Dimension) bool {
	for i := range dim {
		if dim[i].Value == dimensionValueWildcard {
			return true
		}
	}
	return false
}

func compareAWSDimensions(dim1 []cloudwatch.Dimension, dim2 []cloudwatch.Dimension) bool {
	if len(dim1) != len(dim2) {
		return false
	}

	var dim1NameToValue = make(map[string]string, len(dim1))
	var dim2NameToValue = make(map[string]string, len(dim1))

	for i := range dim2 {
		dim1NameToValue[*dim1[i].Name] = *dim1[i].Value
		dim2NameToValue[*dim2[i].Name] = *dim2[i].Value
	}
	for name, v1 := range dim1NameToValue {
		v2, exists := dim2NameToValue[name]
		if exists && v2 == dimensionValueWildcard {
			// wildcard can represent any value, so we set the
			// dimension name with value in CloudWatch ListMetircs result,
			// then the compare result is true
			dim2NameToValue[name] = v1
		}
	}
	return reflect.DeepEqual(dim1NameToValue, dim2NameToValue)
}

func insertTags(events map[string]mb.Event, identifier string, resourceTagMap map[string][]resourcegroupstaggingapi.Tag) {
	// Check if identifier includes dimensionSeparator (comma in this case),
	// split the identifier and check for each sub-identifier.
	// For example, identifier might be [storageType, s3BucketName].
	// And tags are only store under s3BucketName in resourceTagMap.
	subIdentifiers := strings.Split(identifier, dimensionSeparator)
	for _, v := range subIdentifiers {
		tags := resourceTagMap[v]
		// some metric dimension values are arn format, eg: AWS/DDOS namespace metric
		if len(tags) == 0 && strings.HasPrefix(v, "arn:") {
			resourceID, err := aws.FindShortIdentifierFromARN(v)
			if err == nil {
				tags = resourceTagMap[resourceID]
			}
		}
		if len(tags) != 0 {
			// By default, replace dot "." using underscore "_" for tag keys.
			// Note: tag values are not dedotted.
			for _, tag := range tags {
				events[identifier].RootFields.Put("aws.tags."+common.DeDot(*tag.Key), *tag.Value)
			}
			continue
		}
	}
}
