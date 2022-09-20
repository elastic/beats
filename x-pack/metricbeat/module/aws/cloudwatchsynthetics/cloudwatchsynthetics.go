// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudwatchsynthetics

import (
	"fmt"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	resourcegroupstaggingapitypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
	cw "github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/cloudwatch"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	metricsetName      = "cloudwatchsynthetics"
	namespaceIdx       = 1
	identifierValueIdx = 4
	defaultStatistics  = []string{"Average", "Sum"}
	labelSeparator     = "|"
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
	CloudwatchConfigs []cw.Config `config:"metrics" validate:"nonzero,required"`
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return cw.New(base)
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
	if len(listMetricDetailTotal.MetricsWithStats) != 0 {
		for _, regionName := range m.MetricSet.RegionsList {
			//m.logger.Debugf("Collecting metrics from AWS region %s", regionName)
			beatsConfig := m.MetricSet.AwsConfig.Copy()
			beatsConfig.Region = regionName

			svcCloudwatch, svcResourceAPI, err := m.createAwsRequiredClients(beatsConfig, regionName, config)
			if err != nil {
				m.Logger().Warn("skipping metrics list from region '%s'", regionName)
			}

			eventsWithIdentifier, err := m.createEvents(svcCloudwatch, svcResourceAPI, listMetricDetailTotal.MetricsWithStats, listMetricDetailTotal.ResourceTypeFilters, regionName, startTime, endTime)
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
	for _, regionName := range m.MetricSet.RegionsList {
		m.logger.Debugf("Collecting metrics from AWS region %s", regionName)
		beatsConfig := m.MetricSet.AwsConfig.Copy()
		beatsConfig.Region = regionName

		svcCloudwatch, svcResourceAPI, err := m.createAwsRequiredClients(beatsConfig, regionName, config)
		if err != nil {
			m.Logger().Warn("skipping metrics list from region '%s'", regionName)
		}
		for namespace, namespaceDetails := range namespaceDetailTotal {
			m.logger.Debugf("Collected metrics from namespace %s", namespace)

			listMetricsOutput, err := aws.GetListMetricsOutput(namespace, regionName, svcCloudwatch)
			if err != nil {
				m.logger.Info(err.Error())
				continue
			}

			if len(listMetricsOutput) == 0 {
				continue
			}

			// filter listMetricsOutput by detailed configuration per each namespace
			filteredMetricWithStatsTotal := filterListMetricsOutput(listMetricsOutput, namespaceDetails)
			// get resource type filters and tags filters for each namespace
			resourceTypeTagFilters := constructTagsFilters(namespaceDetails)

			eventsWithIdentifier, err := m.createEvents(svcCloudwatch, svcResourceAPI, filteredMetricWithStatsTotal, resourceTypeTagFilters, regionName, startTime, endTime)
			if err != nil {
				return fmt.Errorf("createEvents failed for region %s: %w", regionName, err)
			}

			m.logger.Debugf("Collected number of metrics = %d", len(eventsWithIdentifier))

			for _, event := range eventsWithIdentifier {
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
func filterListMetricsOutput(listMetricsOutput []types.Metric, namespaceDetails []cw.NamespaceDetail) []cw.MetricsWithStatistics {
	return cw.FilterListMetricsOutput(listMetricsOutput, namespaceDetails)
}

// Collect resource type filters and tag filters from config for cloudwatch
func constructTagsFilters(namespaceDetails []cw.NamespaceDetail) map[string][]aws.Tag {
	return cw.ConstructTagsFilters(namespaceDetails)
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

func (m *MetricSet) readCloudwatchConfig() (cw.ListMetricWithDetail, map[string][]cw.NamespaceDetail) {
	var listMetricDetailTotal cw.ListMetricWithDetail
	namespaceDetailTotal := map[string][]cw.NamespaceDetail{}
	var metricsWithStatsTotal []cw.MetricsWithStatistics
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
				metricsWithStats := cw.MetricsWithStatistics{
					CloudwatchMetric: types.Metric{
						Namespace:  &namespace,
						MetricName: &config.MetricName[i],
						Dimensions: cloudwatchDimensions,
					},
					Statistic: config.Statistic,
				}
				metricsWithStatsTotal = append(metricsWithStatsTotal, metricsWithStats)
			}

			if config.ResourceType != "" {
				resourceTypesWithTags[config.ResourceType] = m.MetricSet.TagsFilter
			}
			continue
		}

		configPerNamespace := cw.NamespaceDetail{
			Names:              config.MetricName,
			Tags:               m.MetricSet.TagsFilter,
			Statistics:         config.Statistic,
			ResourceTypeFilter: config.ResourceType,
			Dimensions:         cloudwatchDimensions,
		}

		namespaceDetailTotal[config.Namespace] = append(namespaceDetailTotal[config.Namespace], configPerNamespace)
	}

	listMetricDetailTotal.ResourceTypeFilters = resourceTypesWithTags
	listMetricDetailTotal.MetricsWithStats = metricsWithStatsTotal
	return listMetricDetailTotal, namespaceDetailTotal
}

func createMetricDataQueries(listMetricsTotal []cw.MetricsWithStatistics, period time.Duration) []types.MetricDataQuery {
	return cw.CreateMetricDataQueries(listMetricsTotal, period)
}

func statisticLookup(stat string) (string, bool) {
	return cw.StatisticLookup(stat)
}

func insertRootFields(event mb.Event, metricValue float64, labels []string) mb.Event {
	return cw.InsertRootFields(event, metricValue, labels)
}

func (m *MetricSet) createEvents(svcCloudwatch cloudwatch.GetMetricDataAPIClient, svcResourceAPI resourcegroupstaggingapi.GetResourcesAPIClient, listMetricWithStatsTotal []cw.MetricsWithStatistics, resourceTypeTagFilters map[string][]aws.Tag, regionName string, startTime time.Time, endTime time.Time) (map[string]mb.Event, error) {
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
		return events, fmt.Errorf("getMetricDataResults failed: %w", err)
	}

	// Find a timestamp for all metrics in output
	timestamp := aws.FindTimestamp(metricDataResults)
	if timestamp.IsZero() {
		return nil, nil
	}

	// Create events when there is no tags_filter or resource_type specified.
	if len(resourceTypeTagFilters) == 0 {
		for _, metricDataResult := range metricDataResults {
			if len(metricDataResult.Values) == 0 {
				continue
			}

			exists, timestampIdx := aws.CheckTimestampInArray(timestamp, metricDataResult.Timestamps)
			if exists {
				labels := strings.Split(*metricDataResult.Label, labelSeparator)
				if len(labels) != 5 {
					// when there is no identifier value in label, use region+accountID+namespace instead
					identifier := regionName + m.AccountID + labels[namespaceIdx]
					if _, ok := events[identifier]; !ok {
						events[identifier] = aws.InitEvent(regionName, m.AccountName, m.AccountID, timestamp)
					}
					events[identifier] = insertRootFields(events[identifier], metricDataResult.Values[timestampIdx], labels)
					continue
				}

				identifierValue := labels[identifierValueIdx]
				if _, ok := events[identifierValue]; !ok {
					events[identifierValue] = aws.InitEvent(regionName, m.AccountName, m.AccountID, timestamp)
				}
				events[identifierValue] = insertRootFields(events[identifierValue], metricDataResult.Values[timestampIdx], labels)
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

func configDimensionValueContainsWildcard(dim []cw.Dimension) bool {
	return cw.ConfigDimensionValueContainsWildcard(dim)
}

func insertTags(events map[string]mb.Event, identifier string, resourceTagMap map[string][]resourcegroupstaggingapitypes.Tag) {
	cw.InsertTags(events, identifier, resourceTagMap)
}
