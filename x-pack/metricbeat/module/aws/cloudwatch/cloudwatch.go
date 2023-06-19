// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudwatch

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	resourcegroupstaggingapitypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	metricsetName          = "cloudwatch"
	defaultStatistics      = []string{"Average", "Maximum", "Minimum", "Sum", "SampleCount"}
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
// mb.BaseMetricSet because it implements all the required mb.MetricSet
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
	cloudwatchMetric aws.MetricWithID
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
	dimensions         []types.Dimension
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logger := logp.NewLogger(metricsetName)
	metricSet, err := aws.NewMetricSet(base)
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
		for _, regionName := range m.MetricSet.RegionsList {
			m.logger.Debugf("Collecting metrics from AWS region %s", regionName)
			beatsConfig := m.MetricSet.AwsConfig.Copy()
			beatsConfig.Region = regionName

			svcCloudwatch, svcResourceAPI, err := m.createAwsRequiredClients(beatsConfig, regionName, config)
			if err != nil {
				m.Logger().Warn("skipping metrics list from region '%s'", regionName)
			}

			eventsWithIdentifier, err := m.createEvents(svcCloudwatch, svcResourceAPI, listMetricDetailTotal.metricsWithStats, listMetricDetailTotal.resourceTypeFilters, regionName, startTime, endTime)
			if err != nil {
				return fmt.Errorf("createEvents failed for region %s: %w", regionName, err)
			}

			m.logger.Debugf("Collected metrics of metrics = %d", len(eventsWithIdentifier))

			for _, event := range eventsWithIdentifier {
				_ = event.RootFields.Delete(aws.CloudWatchPeriodName)
				report.Event(event)
			}
		}
	}

	// Create events based on namespaceDetailTotal from configuration
	for _, regionName := range m.MetricSet.RegionsList {
		m.logger.Debugf("Collecting metrics from AWS region %s", regionName)
		beatsConfig := m.MetricSet.AwsConfig.Copy()
		beatsConfig.Region = regionName

		svcCloudwatch, svcResourceAPI, err := m.createAwsRequiredClients(beatsConfig, regionName, config)
		if err != nil {
			m.Logger().Warn("skipping metrics list from region '%s'", regionName, err)
			continue
		}

		// retrieve all the details for all the metrics available in the current region
		listMetricsOutput, err := aws.GetListMetricsOutput("*", regionName, m.Period, m.IncludeLinkedAccounts, m.MonitoringAccountID, svcCloudwatch)
		if err != nil {
			m.Logger().Errorf("Error while retrieving the list of metrics for region %s: %w", regionName, err)
		}

		if len(listMetricsOutput) == 0 {
			continue
		}

		for namespace, namespaceDetails := range namespaceDetailTotal {
			m.logger.Debugf("Collected metrics from namespace %s", namespace)

			// filter listMetricsOutput by detailed configuration per each namespace
			filteredMetricWithStatsTotal := filterListMetricsOutput(listMetricsOutput, namespace, namespaceDetails)

			// get resource type filters and tags filters for each namespace
			resourceTypeTagFilters := constructTagsFilters(namespaceDetails)

			eventsWithIdentifier, err := m.createEvents(svcCloudwatch, svcResourceAPI, filteredMetricWithStatsTotal, resourceTypeTagFilters, regionName, startTime, endTime)
			if err != nil {
				return fmt.Errorf("createEvents failed for region %s: %w", regionName, err)
			}

			m.logger.Debugf("Collected number of metrics = %d", len(eventsWithIdentifier))

			events, err := addMetadata(m.logger, namespace, regionName, beatsConfig, config.AWSConfig.FIPSEnabled, eventsWithIdentifier)
			if err != nil {
				// TODO What to do if add metadata fails? I guess to continue, probably we have an 90% of reliable data
				m.Logger().Warnf("could not add metadata to events: %v", err)
			}

			for _, event := range events {
				_ = event.RootFields.Delete(aws.CloudWatchPeriodName)
				report.Event(event)
			}
		}
	}
	return nil
}

// createAwsRequiredClients will return the two necessary client instances to do Metric requests to the AWS API
func (m *MetricSet) createAwsRequiredClients(beatsConfig awssdk.Config, regionName string, config aws.Config) (*cloudwatch.Client, *resourcegroupstaggingapi.Client, error) {
	m.logger.Debugf("Collecting metrics from AWS region %s", regionName)

	svcCloudwatchClient := cloudwatch.NewFromConfig(beatsConfig, func(o *cloudwatch.Options) {
		if config.AWSConfig.FIPSEnabled {
			o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
		}

	})

	svcResourceAPIClient := resourcegroupstaggingapi.NewFromConfig(beatsConfig, func(o *resourcegroupstaggingapi.Options) {
		if config.AWSConfig.FIPSEnabled {
			o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
		}
	})

	return svcCloudwatchClient, svcResourceAPIClient, nil
}

// filterListMetricsOutput compares config details with listMetricsOutput and filter out the ones don't match
func filterListMetricsOutput(listMetricsOutput []aws.MetricWithID, namespace string, namespaceDetails []namespaceDetail) []metricsWithStatistics {
	var filteredMetricWithStatsTotal []metricsWithStatistics
	for _, listMetric := range listMetricsOutput {
		if *listMetric.Metric.Namespace == namespace {
			for _, configPerNamespace := range namespaceDetails {
				if configPerNamespace.names != nil {
					// Consider only the metrics that exist in the configuration
					exists, _ := aws.StringInSlice(*listMetric.Metric.MetricName, configPerNamespace.names)
					if !exists {
						continue
					}
				}
				if configPerNamespace.dimensions != nil {
					if !compareAWSDimensions(listMetric.Metric.Dimensions, configPerNamespace.dimensions) {
						continue
					}
				}
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

func (m *MetricSet) checkStatistics() error {
	for _, config := range m.CloudwatchConfigs {
		for _, stat := range config.Statistic {
			if _, ok := statisticLookup(stat); !ok {
				return fmt.Errorf("statistic method specified is not valid: %s", stat)
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

		var cloudwatchDimensions []types.Dimension
		for _, dim := range config.Dimensions {
			name := dim.Name
			value := dim.Value
			cloudwatchDimensions = append(cloudwatchDimensions, types.Dimension{
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
				metric := types.Metric{
					Namespace:  &namespace,
					MetricName: &config.MetricName[i],
					Dimensions: cloudwatchDimensions,
				}

				metricsWithStats := metricsWithStatistics{
					cloudwatchMetric: aws.MetricWithID{
						Metric: metric,
					},
					statistic: config.Statistic,
				}

				metricsWithStatsTotal = append(metricsWithStatsTotal, metricsWithStats)
			}

			if config.ResourceType != "" {
				resourceTypesWithTags[config.ResourceType] = m.MetricSet.TagsFilter
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

		namespaceDetailTotal[config.Namespace] = append(namespaceDetailTotal[config.Namespace], configPerNamespace)
	}

	listMetricDetailTotal.resourceTypeFilters = resourceTypesWithTags
	listMetricDetailTotal.metricsWithStats = metricsWithStatsTotal
	return listMetricDetailTotal, namespaceDetailTotal
}

func createMetricDataQueries(listMetricsTotal []metricsWithStatistics, dataGranularity time.Duration) []types.MetricDataQuery {
	var metricDataQueries []types.MetricDataQuery
	for i, listMetric := range listMetricsTotal {
		for j, statistic := range listMetric.statistic {
			stat := statistic
			metric := listMetric.cloudwatchMetric
			label := constructLabel(listMetric.cloudwatchMetric, statistic)
			dataGranularityInSec := int32(dataGranularity.Seconds())

			id := "cw" + strconv.Itoa(i) + "stats" + strconv.Itoa(j)
			metricDataQuery := types.MetricDataQuery{
				Id: &id,
				MetricStat: &types.MetricStat{
					Period: &dataGranularityInSec,
					Stat:   &stat,
					Metric: &metric.Metric,
				},
				Label: &label,
			}
			if listMetric.cloudwatchMetric.AccountID != "" {
				metricDataQuery.AccountId = &metric.AccountID
			}

			metricDataQueries = append(metricDataQueries, metricDataQuery)
		}
	}
	return metricDataQueries
}

func constructLabel(metric aws.MetricWithID, statistic string) string {
	// label = accountID + accountLabel + metricName + namespace + statistic + periodLabel + dimKeys + dimValues
	label := strings.Join([]string{metric.AccountID, aws.LabelConst.AccountLabel, *metric.Metric.MetricName, *metric.Metric.Namespace, statistic, aws.LabelConst.PeriodLabel}, aws.LabelConst.LabelSeparator)
	dimNames := ""
	dimValues := ""
	for i, dim := range metric.Metric.Dimensions {
		dimNames += *dim.Name
		dimValues += *dim.Value
		if i != len(metric.Metric.Dimensions)-1 {
			dimNames += dimensionSeparator
			dimValues += dimensionSeparator
		}
	}

	if dimNames != "" && dimValues != "" {
		label += aws.LabelConst.LabelSeparator + dimNames
		label += aws.LabelConst.LabelSeparator + dimValues
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
	stat := labels[aws.LabelConst.StatisticIdx]
	// Check if statistic method is one of Sum, SampleCount, Minimum, Maximum, Average
	// With checkStatistics function, no need to check bool return value here
	statMethod, _ := statisticLookup(stat)
	// By default, replace dot "." using underscore "_" for metric names
	return "aws." + stripNamespace(namespace) + ".metrics." + common.DeDot(labels[aws.LabelConst.MetricNameIdx]) + "." + statMethod
}

// stripNamespace converts Cloudwatch namespace into the root field we will use for metrics
// example AWS/EC2 -> ec2
func stripNamespace(namespace string) string {
	parts := strings.Split(namespace, "/")
	return strings.ToLower(parts[len(parts)-1])
}

func insertRootFields(event mb.Event, metricValue float64, labels []string) mb.Event {
	namespace := labels[aws.LabelConst.NamespaceIdx]
	_, _ = event.RootFields.Put(generateFieldName(namespace, labels), metricValue)
	_, _ = event.RootFields.Put("aws.cloudwatch.namespace", namespace)
	if len(labels) != aws.LabelConst.LabelLengthTotal {
		return event
	}

	dimNames := strings.Split(labels[aws.LabelConst.IdentifierNameIdx], ",")
	dimValues := strings.Split(labels[aws.LabelConst.IdentifierValueIdx], ",")
	for i := 0; i < len(dimNames); i++ {
		_, _ = event.RootFields.Put("aws.dimensions."+dimNames[i], dimValues[i])
	}
	return event
}

func (m *MetricSet) createEvents(svcCloudwatch cloudwatch.GetMetricDataAPIClient, svcResourceAPI resourcegroupstaggingapi.GetResourcesAPIClient, listMetricWithStatsTotal []metricsWithStatistics, resourceTypeTagFilters map[string][]aws.Tag, regionName string, startTime time.Time, endTime time.Time) (map[string]mb.Event, error) {
	// Initialize events for each identifier.
	events := make(map[string]mb.Event)

	// Construct metricDataQueries
	metricDataQueries := createMetricDataQueries(listMetricWithStatsTotal, m.DataGranularity)
	m.logger.Debugf("Number of MetricDataQueries = %d", len(metricDataQueries))
	if len(metricDataQueries) == 0 {
		return events, nil
	}

	// Use metricDataQueries to make GetMetricData API calls
	metricDataResults, err := aws.GetMetricDataResults(metricDataQueries, svcCloudwatch, startTime, endTime)
	m.logger.Debugf("Number of metricDataResults = %d", len(metricDataResults))
	if err != nil {
		return events, fmt.Errorf("getMetricDataResults failed: %w", err)
	}

	// Create events when there is no tags_filter or resource_type specified.
	if len(resourceTypeTagFilters) == 0 {
		for _, metricDataResult := range metricDataResults {
			if len(metricDataResult.Values) == 0 {
				continue
			}
			labels := strings.Split(*metricDataResult.Label, aws.LabelConst.LabelSeparator)
			for valI, metricDataResultValue := range metricDataResult.Values {
				if len(labels) != aws.LabelConst.LabelLengthTotal {
					// when there is no identifier value in label, use id+label+region+accountID+namespace+index instead
					identifier := labels[aws.LabelConst.AccountIdIdx] + labels[aws.LabelConst.AccountLabelIdx] + regionName + m.MonitoringAccountID + labels[aws.LabelConst.NamespaceIdx] + fmt.Sprint("-", valI)
					if _, ok := events[identifier]; !ok {
						if labels[aws.LabelConst.AccountIdIdx] != "" {
							events[identifier] = aws.InitEvent(regionName, labels[aws.LabelConst.AccountLabelIdx], labels[aws.LabelConst.AccountIdIdx], metricDataResult.Timestamps[valI], labels[aws.LabelConst.PeriodLabelIdx])
						} else {
							events[identifier] = aws.InitEvent(regionName, m.MonitoringAccountName, m.MonitoringAccountID, metricDataResult.Timestamps[valI], labels[aws.LabelConst.PeriodLabelIdx])
						}
					}
					events[identifier] = insertRootFields(events[identifier], metricDataResultValue, labels)
					continue
				}

				identifierValue := labels[aws.LabelConst.IdentifierValueIdx] + fmt.Sprint("-", valI)
				if _, ok := events[identifierValue]; !ok {
					events[identifierValue] = aws.InitEvent(regionName, labels[aws.LabelConst.AccountLabelIdx], labels[aws.LabelConst.AccountIdIdx], metricDataResult.Timestamps[valI], labels[aws.LabelConst.PeriodLabelIdx])
				}
				events[identifierValue] = insertRootFields(events[identifierValue], metricDataResultValue, labels)
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

		for _, output := range metricDataResults {
			if len(output.Values) == 0 {
				continue
			}

			labels := strings.Split(*output.Label, aws.LabelConst.LabelSeparator)
			for valI, metricDataResultValue := range output.Values {
				if len(labels) != aws.LabelConst.LabelLengthTotal {
					// if there is no tag in labels but there is a tagsFilter, then no event should be reported.
					if len(tagsFilter) != 0 {
						continue
					}

					// when there is no identifier value in label, use id+label+region+accountID+namespace+index instead
					identifier := labels[aws.LabelConst.AccountIdIdx] + labels[aws.LabelConst.AccountLabelIdx] + regionName + m.MonitoringAccountID + labels[aws.LabelConst.NamespaceIdx] + fmt.Sprint("-", valI)
					if _, ok := events[identifier]; !ok {
						if labels[aws.LabelConst.AccountIdIdx] != "" {
							events[identifier] = aws.InitEvent(regionName, labels[aws.LabelConst.AccountLabelIdx], labels[aws.LabelConst.AccountIdIdx], output.Timestamps[valI], labels[aws.LabelConst.PeriodLabelIdx])
						} else {
							events[identifier] = aws.InitEvent(regionName, m.MonitoringAccountName, m.MonitoringAccountID, output.Timestamps[valI], labels[aws.LabelConst.PeriodLabelIdx])
						}
					}
					events[identifier] = insertRootFields(events[identifier], metricDataResultValue, labels)
					continue
				}

				identifierValue := labels[aws.LabelConst.IdentifierValueIdx]
				uniqueIdentifierValue := identifierValue + fmt.Sprint("-", valI)

				// add tags to event based on identifierValue
				// Check if identifier includes dimensionSeparator (comma in this case),
				// split the identifier and check for each sub-identifier.
				// For example, identifier might be [storageType, s3BucketName].
				// And tags are only store under s3BucketName in resourceTagMap.
				subIdentifiers := strings.Split(identifierValue, dimensionSeparator)
				for _, subIdentifier := range subIdentifiers {
					if _, ok := events[uniqueIdentifierValue]; !ok {
						// when tagsFilter is not empty but no entry in
						// resourceTagMap for this identifier, do not initialize
						// an event for this identifier.
						if len(tagsFilter) != 0 && resourceTagMap[subIdentifier] == nil {
							continue
						}
						events[uniqueIdentifierValue] = aws.InitEvent(regionName, labels[aws.LabelConst.AccountLabelIdx], labels[aws.LabelConst.AccountIdIdx], output.Timestamps[valI], labels[aws.LabelConst.PeriodLabelIdx])
					}
					events[uniqueIdentifierValue] = insertRootFields(events[uniqueIdentifierValue], metricDataResultValue, labels)
					insertTags(events, uniqueIdentifierValue, subIdentifier, resourceTagMap)
				}
			}
		}
	}
	return events, nil
}

func configDimensionValueContainsWildcard(dim []Dimension) bool {
	for i := range dim {
		if dim[i].Value == dimensionValueWildcard {
			return true
		}
	}
	return false
}

func compareAWSDimensions(dim1 []types.Dimension, dim2 []types.Dimension) bool {
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

func insertTags(events map[string]mb.Event, uniqueIdentifierValue string, subIdentifier string, resourceTagMap map[string][]resourcegroupstaggingapitypes.Tag) {
	tags := resourceTagMap[subIdentifier]
	// some metric dimension values are arn format, eg: AWS/DDOS namespace metric
	if len(tags) == 0 && strings.HasPrefix(subIdentifier, "arn:") {
		resourceID, err := aws.FindShortIdentifierFromARN(subIdentifier)
		if err == nil {
			tags = resourceTagMap[resourceID]
		}
	}
	if len(tags) != 0 {
		// By default, replace dot "." using underscore "_" for tag keys.
		// Note: tag values are not dedotted.
		for _, tag := range tags {
			_, _ = events[uniqueIdentifierValue].RootFields.Put("aws.tags."+common.DeDot(*tag.Key), *tag.Value)
		}
	}
}
