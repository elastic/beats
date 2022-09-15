// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudwatchsynthetics

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
	cw "github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/cloudwatch"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/awsV1"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	metricsetName          = "cloudwatchsynthetics"
	defaultStatistics      = []string{"Average", "Sum"}
	dimensionValueWildcard = "*"
	labelSeparator         = "|"
	dimensionSeparator     = ","
	namespaceIdx           = 1
	identifierValueIdx     = 4
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("awsV1", "cloudwatchsynthetics", New)
}

type metricsWithStatistics struct {
	cloudwatchMetric cloudwatch.Metric
	statistic        []string
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

// Config holds a configuration specific for cloudwatch metricset.
type Config struct {
	Namespace    string                 `config:"namespace" validate:"nonzero,required"`
	MetricName   []string               `config:"name"`
	Dimensions   []cloudwatch.Dimension `config:"dimensions"`
	ResourceType string                 `config:"resource_type"`
	Statistic    []string               `config:"statistic"`
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	RegionsList       []string
	Endpoint          string
	Period            time.Duration
	Latency           time.Duration
	AwsConfig         *awssdk.Config
	AccountName       string
	AccountID         string
	TagsFilter        []aws.Tag
	logger            *logp.Logger
	CloudwatchConfigs []Config `config:"metrics" validate:"nonzero,required"`
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logger := logp.NewLogger(metricsetName)
	metricSet, err := awsV1.NewMetricSet(base)
	if err != nil {
		return nil, fmt.Errorf("error creating aws metricset: %w", err)
	}

	config := struct {
		CloudwatchMetrics []Config `config:"metrics" validate:"nonzero,required"`
	}{}

	err = base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, fmt.Errorf("error unpack raw module config using UnpackConfig: %w", err)
	}

	logger.Debugf("cloudwatch config = %s", config)
	if len(config.CloudwatchMetrics) == 0 {
		return nil, fmt.Errorf("metrics in config is missing: %w", err)
	}

	return &MetricSet{
		BaseMetricSet:     metricSet.BaseMetricSet,
		RegionsList:       metricSet.RegionsList,
		Endpoint:          metricSet.Endpoint,
		Period:            metricSet.Period,
		Latency:           metricSet.Latency,
		AwsConfig:         metricSet.AwsConfig,
		AccountName:       metricSet.AccountName,
		AccountID:         metricSet.AccountID,
		TagsFilter:        metricSet.TagsFilter,
		logger:            logger,
		CloudwatchConfigs: config.CloudwatchMetrics,
	}, nil
}

func (m *MetricSet) checkStatistics() error {
	for _, config := range m.CloudwatchConfigs {
		for _, stat := range config.Statistic {
			if _, ok := cw.StatisticLookup(stat); !ok {
				return fmt.Errorf("statistic method specified is not valid: %s", stat)
			}
		}
	}
	return nil
}

func configDimensionValueContainsWildcard(dim []cloudwatch.Dimension) bool {
	for i := range dim {
		if *dim[i].Value == dimensionValueWildcard {
			return true
		}
	}
	return false
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
				Name:  name,
				Value: value,
			})
		}

		cloudwatchDimensionsV1 := awsV1.PointersOf(cloudwatchDimensions).([]*cloudwatch.Dimension)

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
						Dimensions: cloudwatchDimensionsV1,
					},
					statistic: config.Statistic,
				}
				metricsWithStatsTotal = append(metricsWithStatsTotal, metricsWithStats)
			}

			if config.ResourceType != "" {
				resourceTypesWithTags[config.ResourceType] = m.TagsFilter
			}
			continue
		}

		configPerNamespace := namespaceDetail{
			names:              config.MetricName,
			tags:               m.TagsFilter,
			statistics:         config.Statistic,
			resourceTypeFilter: config.ResourceType,
			dimensions:         cloudwatchDimensions,
		}

		namespaceDetailTotal[config.Namespace] = append(namespaceDetailTotal[config.Namespace], configPerNamespace)
	}

	listMetricDetailTotal.resourceTypeFilters = resourceTypesWithTags
	listMetricDetailTotal.metricsWithStats = metricsWithStatsTotal
	return listMetricDetailTotal, namespaceDetailTotal
}

// createAwsRequiredClients will return the two necessary client instances to do Metric requests to the AWS API
func (m *MetricSet) createAwsRequiredClients(beatsConfig awssdk.Config, regionName string, config aws.Config) (*cloudwatch.CloudWatch, *resourcegroupstaggingapi.ResourceGroupsTaggingAPI, error) {
	m.logger.Debugf("Collecting metrics from AWS region %s", regionName)

	mySession := session.Must(session.NewSession())
	svcCloudwatchSyntheticsClient := cloudwatch.New(mySession, &beatsConfig)

	svcResourceAPIClient := resourcegroupstaggingapi.New(mySession, &beatsConfig)

	return svcCloudwatchSyntheticsClient, svcResourceAPIClient, nil
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

// filterListMetricsOutput compares config details with listMetricsOutput and filter out the ones don't match
func filterListMetricsOutput(listMetricsOutput []cloudwatch.Metric, namespaceDetails []namespaceDetail) []metricsWithStatistics {
	var filteredMetricWithStatsTotal []metricsWithStatistics
	for _, listMetric := range listMetricsOutput {
		listMetricV1 := awsV1.DereferenceArr(listMetric.Dimensions).([]cloudwatch.Dimension)
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
					})

			} else if configPerNamespace.names == nil && configPerNamespace.dimensions != nil {
				// if metric names are not given in config but dimensions are
				// given, only keep the metrics with matching dimensions
				if !compareAWSDimensions(listMetricV1, configPerNamespace.dimensions) {
					continue
				}
				filteredMetricWithStatsTotal = append(filteredMetricWithStatsTotal,
					metricsWithStatistics{
						cloudwatchMetric: listMetric,
						statistic:        configPerNamespace.statistics,
					})
			} else if configPerNamespace.names != nil && configPerNamespace.dimensions != nil {
				if exists, _ := aws.StringInSlice(*listMetric.MetricName, configPerNamespace.names); !exists {
					continue
				}
				if !compareAWSDimensions(listMetricV1, configPerNamespace.dimensions) {
					continue
				}
				filteredMetricWithStatsTotal = append(filteredMetricWithStatsTotal,
					metricsWithStatistics{
						cloudwatchMetric: listMetric,
						statistic:        configPerNamespace.statistics,
					})
			} else {
				// if no metric name and no dimensions given, then keep all listMetricsOutput
				filteredMetricWithStatsTotal = append(filteredMetricWithStatsTotal,
					metricsWithStatistics{
						cloudwatchMetric: listMetric,
						statistic:        configPerNamespace.statistics,
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

// addMetadata adds metadata to the given events map based on namespace
func addMetadata(namespace string, regionName string, awsConfig awssdk.Config, fipsEnabled bool, events map[string]mb.Event) (map[string]mb.Event, error) {
	return events, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// Get startTime and endTime
	startTime, endTime := aws.GetStartTimeEndTime(time.Now(), m.Period, m.Latency)
	m.Logger().Debugf("startTime = %s, endTime = %s", startTime, endTime)

	// Check statistic method in config
	err := m.checkStatistics()
	if err != nil {
		return fmt.Errorf("checkStatistics failed: %w", err)
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
		for i, regionName := range m.RegionsList {
			//m.logger.Debugf("Collecting metrics from AWS region %s", regionName)
			beatsConfig := m.AwsConfig.Copy()
			beatsConfig.Region = &m.RegionsList[i]

			svcCloudwatch, svcResourceAPI, err := m.createAwsRequiredClients(*beatsConfig, regionName, config)
			if err != nil {
				m.Logger().Warn("skipping metrics list from region '%s'", regionName)
			}

			eventsWithIdentifier, err := m.createEvents(*svcCloudwatch, *svcResourceAPI, listMetricDetailTotal.metricsWithStats, listMetricDetailTotal.resourceTypeFilters, regionName, startTime, endTime)
			if err != nil {
				return fmt.Errorf("createEvents failed for region %s: %w", regionName, err)
			}

			m.logger.Debugf("Collected metrics of metrics = %d", len(eventsWithIdentifier))

			for _, event := range eventsWithIdentifier {
				report.Event(event)
			}
		}
	}

	// Create events based on namespaceDetailTotal from configuration
	for i, regionName := range m.RegionsList {
		m.logger.Debugf("Collecting metrics from AWS region %s", regionName)
		beatsConfig := m.AwsConfig.Copy()
		beatsConfig.Region = &m.RegionsList[i]

		svcCloudwatch, svcResourceAPI, err := m.createAwsRequiredClients(*beatsConfig, regionName, config)
		if err != nil {
			m.Logger().Warn("skipping metrics list from region '%s'", regionName)
		}
		for namespace, namespaceDetails := range namespaceDetailTotal {
			m.logger.Debugf("Collected metrics from namespace %s", namespace)

			cloudWatchSynthetics := "CloudWatchSynthetics"
			listMetricsOutput, err := svcCloudwatch.ListMetrics(&cloudwatch.ListMetricsInput{
				Namespace: &cloudWatchSynthetics,
			})
			if err != nil {
				m.logger.Info(err.Error())
				continue
			}

			if len(listMetricsOutput.Metrics) == 0 {
				continue
			}

			// filter listMetricsOutput by detailed configuration per each namespace
			filteredMetricWithStatsTotal := filterListMetricsOutput(awsV1.DereferenceArr(listMetricsOutput.Metrics).([]cloudwatch.Metric), namespaceDetails)
			// get resource type filters and tags filters for each namespace
			resourceTypeTagFilters := constructTagsFilters(namespaceDetails)

			eventsWithIdentifier, err := m.createEvents(*svcCloudwatch, *svcResourceAPI, filteredMetricWithStatsTotal, resourceTypeTagFilters, regionName, startTime, endTime)
			if err != nil {
				return fmt.Errorf("createEvents failed for region %s: %w", regionName, err)
			}

			m.logger.Debugf("Collected number of metrics = %d", len(eventsWithIdentifier))

			events, err := addMetadata(namespace, regionName, *beatsConfig, config.AWSConfig.FIPSEnabled, eventsWithIdentifier)
			if err != nil {
				// TODO What to do if add metadata fails? I guess to continue, probably we have an 90% of reliable data
				m.Logger().Warn("could not add metadata to events: %w", err)
			}

			for _, event := range events {
				report.Event(event)
			}
		}
	}
	return nil
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

func createMetricDataQueries(listMetricsTotal []metricsWithStatistics, period time.Duration) []cloudwatch.MetricDataQuery {
	var metricDataQueries []cloudwatch.MetricDataQuery
	for i, listMetric := range listMetricsTotal {
		for j, statistic := range listMetric.statistic {
			stat := statistic
			metric := listMetric.cloudwatchMetric
			label := constructLabel(listMetric.cloudwatchMetric, statistic)
			periodInSec := int32(period.Seconds())

			id := "cw" + strconv.Itoa(i) + "stats" + strconv.Itoa(j)
			periodInSec64 := int64(periodInSec)
			metricDataQueries = append(metricDataQueries, cloudwatch.MetricDataQuery{
				Id: &id,
				MetricStat: &cloudwatch.MetricStat{
					Period: &periodInSec64,
					Stat:   &stat,
					Metric: &metric,
				},
				Label: &label,
			})
		}
	}
	return metricDataQueries
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
				_, _ = events[identifier].RootFields.Put("aws.tags."+common.DeDot(*tag.Key), *tag.Value)
			}
			continue
		}
	}
}

func paginate(x []cloudwatch.MetricDataQuery, skip int, size int) []cloudwatch.MetricDataQuery {
	if skip > len(x) {
		skip = len(x)
	}

	end := skip + size
	if end > len(x) {
		end = len(x)
	}

	return x[skip:end]
}

func (m *MetricSet) createEvents(svcCloudwatch cloudwatch.CloudWatch, svcResourceAPI resourcegroupstaggingapi.ResourceGroupsTaggingAPI, listMetricWithStatsTotal []metricsWithStatistics, resourceTypeTagFilters map[string][]aws.Tag, regionName string, startTime time.Time, endTime time.Time) (map[string]mb.Event, error) {
	// Initialize events for each identifier.
	events := map[string]mb.Event{}

	// Construct metricDataQueries
	metricDataQueries := createMetricDataQueries(listMetricWithStatsTotal, m.Period)

	index := 0
	for queries := paginate(metricDataQueries, 0, 500); len(queries) > 0; queries = paginate(metricDataQueries, index, 500) {
		m.logger.Debugf("Number of MetricDataQueries = %d", len(queries))

		metricDataQueriesV1 := awsV1.PointersOf(queries).([]*cloudwatch.MetricDataQuery)

		// Use metricDataQueries to make GetMetricData API calls
		metricDataResults, err := svcCloudwatch.GetMetricData(&cloudwatch.GetMetricDataInput{
			MetricDataQueries: metricDataQueriesV1,
			StartTime:         &startTime,
			EndTime:           &endTime,
		})
		m.logger.Debugf("Number of metricDataResults = %d", len(metricDataResults.MetricDataResults))
		if err != nil {
			return events, fmt.Errorf("getMetricDataResults failed: %w", err)
		}

		metricDataResultsV1 := awsV1.DereferenceArr(metricDataResults.MetricDataResults).([]cloudwatch.MetricDataResult)

		// Find a timestamp for all metrics in output
		timestamp := awsV1.FindTimestamp(metricDataResultsV1)
		if timestamp.IsZero() {
			continue
		}

		// Create events when there is no tags_filter or resource_type specified.
		if len(resourceTypeTagFilters) == 0 {
			for _, metricDataResult := range metricDataResultsV1 {
				if len(metricDataResult.Values) == 0 {
					continue
				}

				metricDataResultTimestampsV1 := awsV1.DereferenceArr(metricDataResult.Timestamps).([]time.Time)

				exists, timestampIdx := aws.CheckTimestampInArray(timestamp, metricDataResultTimestampsV1)
				if exists {
					labels := strings.Split(*metricDataResult.Label, labelSeparator)
					if len(labels) != 5 {
						// when there is no identifier value in label, use region+accountID+namespace instead
						identifier := regionName + m.AccountID + labels[namespaceIdx]
						if _, ok := events[identifier]; !ok {
							events[identifier] = aws.InitEvent(regionName, m.AccountName, m.AccountID, timestamp)
						}
						events[identifier] = cw.InsertRootFields(events[identifier], *metricDataResult.Values[timestampIdx], labels)
						continue
					}

					identifierValue := labels[identifierValueIdx]
					if _, ok := events[identifierValue]; !ok {
						events[identifierValue] = aws.InitEvent(regionName, m.AccountName, m.AccountID, timestamp)
					}
					events[identifierValue] = cw.InsertRootFields(events[identifierValue], *metricDataResult.Values[timestampIdx], labels)
				}
			}
			// return events, nil
		}

		// Create events with tags
		for resourceType, tagsFilter := range resourceTypeTagFilters {
			m.logger.Debugf("resourceType = %s", resourceType)
			m.logger.Debugf("tagsFilter = %s", tagsFilter)
			resourceTagMap, err := awsV1.GetResourcesTags(svcResourceAPI, []string{resourceType})
			if err != nil {
				// If GetResourcesTags failed, continue report event just without tags.
				m.logger.Info(fmt.Errorf("getResourcesTags failed, skipping region %s: %w", regionName, err))
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

			for _, output := range metricDataResults.MetricDataResults {
				if len(output.Values) == 0 {
					continue
				}

				exists, timestampIdx := aws.CheckTimestampInArray(timestamp, awsV1.DereferenceArr(output.Timestamps).([]time.Time))
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
						events[identifier] = cw.InsertRootFields(events[identifier], *output.Values[timestampIdx], labels)
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
					events[identifierValue] = cw.InsertRootFields(events[identifierValue], *output.Values[timestampIdx], labels)

					// add tags to event based on identifierValue
					insertTags(events, identifierValue, resourceTagMap)
				}
			}
		}
		index += 500
	}
	return events, nil
}
