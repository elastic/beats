// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package autoscaling

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	autoscalingTypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cloudwatchTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	resourcegroupstaggingapiTypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/pkg/errors"

	"strconv"
	"strings"
	"time"
)

var (
	metricsetName                  = "autoscaling"
	dimensionNameIdx               = 0
	dimensionValueIdx              = 1
	metricNameIdx                  = 2
	statisticIdx                   = 3
	labelSeparator                 = "|"
	cloudWatchNameSpaceAutoScaling = "AWS/AutoScaling"
)

var defaultMetricStats = map[string][]string{
	"GroupMaxSize":         {"Minimum"},
	"GroupMinSize":         {"Minimum"},
	"GroupDesiredCapacity": {"Minimum"},
	//capacity
	"GroupInServiceCapacity":   {"Minimum"},
	"GroupStandbyCapacity":     {"Minimum"},
	"GroupPendingCapacity":     {"Minimum"},
	"GroupTerminatingCapacity": {"Maximum"},
	"GroupTotalCapacity":       {"Maximum"},
	//instance
	"GroupInServiceInstances":   {"Minimum"},
	"GroupStandbyInstances":     {"Minimum"},
	"GroupPendingInstances":     {"Minimum"},
	"GroupTerminatingInstances": {"Maximum"},
	"GroupTotalInstances":       {"Maximum"},
	//warm pool
	"WarmPoolMinSize":                 {"Minimum"},
	"WarmPoolDesiredCapacity":         {"Minimum"},
	"WarmPoolPendingCapacity":         {"Minimum"},
	"WarmPoolTerminatingCapacity":     {"Maximum"},
	"WarmPoolWarmedCapacity":          {"Maximum"},
	"WarmPoolTotalCapacity":           {"Maximum"},
	"GroupAndWarmPoolDesiredCapacity": {"Maximum"},
	"GroupAndWarmPoolTotalCapacity":   {"Maximum"},
	//predictive scaling.
	"PredictiveScalingLoadForecast":     {"Minimum"},
	"PredictiveScalingCapacityForecast": {"Minimum"},
}

var statisticConvertTable = map[string]string{
	"Average":     "avg",
	"Sum":         "sum",
	"Maximum":     "max",
	"Minimum":     "min",
	"SampleCount": "count",
}

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
	logger        *logp.Logger
	MetricConfigs []MetricConfigs `config:"metrics"`
}

// MetricConfigs holds a configuration specific for AutoScalingGroup metric.
type MetricConfigs struct {
	MetricName []string `config:"name"`
	Statistic  []string `config:"statistic"`
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}

	config := struct {
		MetricConfigs []MetricConfigs `config:"metrics"`
	}{}

	err = base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, errors.Wrap(err, "error unpack raw module config using UnpackConfig")
	}
	base.Logger().Debugf("autoscaling metricset config = %s", config)
	// Check if period is set to be multiple of 60s
	remainder60 := int(metricSet.Period.Seconds()) % 60
	if remainder60 != 0 {
		err := errors.New("period needs to be set to 60s (or a multiple of 60s)")
		base.Logger().Info(err)
	}

	return &MetricSet{
		MetricSet:     metricSet,
		MetricConfigs: config.MetricConfigs,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	startTime, endTime := aws.GetStartTimeEndTime(m.Period, m.Latency)
	m.Logger().Debugf("startTime = %s, endTime = %s", startTime, endTime)
	var metricStats map[string][]string
	if m.MetricConfigs != nil && len(m.MetricConfigs) > 0 {
		metricStats = convertConfigMetricStats(m.MetricConfigs)
	} else {
		metricStats = defaultMetricStats
	}
	for _, region := range m.MetricSet.RegionsList {
		err := m.fetchMetricsInRegion(metricStats, report, region, startTime, endTime)
		if err != nil {
			m.Logger().Errorf("error occurs when fetching metrics in region %s, %v", region, err)
		}
	}

	return nil
}

func (m *MetricSet) fetchMetricsInRegion(metricStats map[string][]string, report mb.ReporterV2, regionName string, startTime, endTime time.Time) error {
	awsConfig := m.MetricSet.AwsConfig.Copy()
	awsConfig.Region = regionName

	svcCloudwatch := cloudwatch.NewFromConfig(awscommon.EnrichAWSConfigWithEndpoint(
		m.Endpoint, "monitoring", regionName, awsConfig))
	svcASG := autoscaling.NewFromConfig(awscommon.EnrichAWSConfigWithEndpoint(
		m.Endpoint, "autoscaling", regionName, awsConfig))

	autoScalingGroupOutputs, err := getAutoScalingGroupsPerRegion(svcASG)
	if err != nil {
		err = errors.Wrap(err, "failed to get autoscaling groups in region "+regionName)
		m.Logger().Errorf(err.Error())
		report.Error(err)
		return err
	}
	autoScalingGroupTags := getAutoScalingGroupTags(autoScalingGroupOutputs)
	listMetricsOutput := m.getAutoScalingGroupMetrics(svcCloudwatch, regionName, autoScalingGroupTags)
	metricDataQueries := m.generateMetricDataQueries(listMetricsOutput, m.Period, metricStats)
	if len(metricDataQueries) == 0 {
		return nil
	}

	metricDataOutput, err := aws.GetMetricDataResults(metricDataQueries, svcCloudwatch, startTime, endTime)
	if err != nil {
		err = errors.Wrap(err, "GetMetricDataResults failed in region "+regionName)
		m.Logger().Error(err.Error())
		report.Error(err)
		return err
	}
	m.Logger().Debugf("metricDataOutput %v", metricDataOutput)
	events, err := m.createCloudWatchEvents(metricDataOutput, autoScalingGroupOutputs, regionName)
	if err != nil {
		m.Logger().Error(err.Error())
		report.Error(err)
		return err
	}

	for _, event := range events {
		if len(event.RootFields) != 0 {
			if reported := report.Event(event); !reported {
				m.Logger().Debug("Fetch interrupted, failed to emit event")
				continue
			}
		}
	}

	return nil
}

func (m *MetricSet) getAutoScalingGroupMetrics(client *cloudwatch.Client, regionName string, autoScalingGroupTags map[string][]resourcegroupstaggingapiTypes.Tag) []cloudwatchTypes.Metric {
	listMetricsOutput, err := aws.GetListMetricsOutput(cloudWatchNameSpaceAutoScaling, regionName, client)
	if err != nil {
		m.Logger().Error(err.Error())
		return nil
	}
	if listMetricsOutput == nil || len(listMetricsOutput) == 0 {
		return nil
	}
	return m.filterMetricsWithTagsFilter(listMetricsOutput, autoScalingGroupTags)
}

func (m *MetricSet) filterMetricsWithTagsFilter(listMetricsOutput []cloudwatchTypes.Metric, autoScalingGroupTags map[string][]resourcegroupstaggingapiTypes.Tag) []cloudwatchTypes.Metric {
	var filteredMetrics []cloudwatchTypes.Metric
	for _, metric := range listMetricsOutput {
		if len(metric.Dimensions) != 1 {
			continue
		}
		// autoscaling group metrics have only one dimension AutoScalingGroupName
		autoScalingGroupName := *metric.Dimensions[0].Value
		tags := autoScalingGroupTags[autoScalingGroupName]
		if len(tags) != 0 && len(m.TagsFilter) != 0 {
			if exists := aws.CheckTagFiltersExist(m.TagsFilter, tags); !exists {
				m.Logger().Debugf("AutoScalingGroup %v tag doesn't match tags_filter %v", tags, m.TagsFilter)
				continue
			}
		}
		filteredMetrics = append(filteredMetrics, metric)
	}
	return filteredMetrics

}

func (m *MetricSet) createCloudWatchEvents(getMetricDataResults []cloudwatchTypes.MetricDataResult, autoScalingGroupOutputs []autoscalingTypes.AutoScalingGroup, regionName string) (map[string]mb.Event, error) {
	// Find a timestamp for all metrics in output
	timestamp := aws.FindTimestamp(getMetricDataResults)
	if timestamp.IsZero() {
		return nil, nil
	}

	autoScalingGroupName2Detail := make(map[string]autoscalingTypes.AutoScalingGroup)
	for _, g := range autoScalingGroupOutputs {
		autoScalingGroupName2Detail[*g.AutoScalingGroupName] = g
	}
	// Initialize events and metricSetFieldResults per autoScalingName
	events := map[string]mb.Event{}
	metricSetFieldResults := map[string]map[string]interface{}{}
	for autoScalingGroupName := range autoScalingGroupName2Detail {
		events[autoScalingGroupName] = aws.InitEvent(regionName, m.AccountName, m.AccountID, timestamp)
		metricSetFieldResults[autoScalingGroupName] = map[string]interface{}{}
	}

	m.Logger().Debug("getMetricDataResults: %v", getMetricDataResults)

	for _, output := range getMetricDataResults {
		if len(output.Values) == 0 {
			continue
		}

		exists, timestampIdx := aws.CheckTimestampInArray(timestamp, output.Timestamps)
		if exists {
			labels := strings.Split(*output.Label, labelSeparator)
			autoScalingGroupName := labels[dimensionValueIdx]
			dimensionName := labels[dimensionNameIdx]
			stat := statisticConvertTable[labels[statisticIdx]]
			metricName := labels[metricNameIdx]

			// Add tags
			// By default, replace dot "." using under bar "_" for tag keys and values
			for _, tag := range autoScalingGroupName2Detail[autoScalingGroupName].Tags {
				events[autoScalingGroupName].RootFields.Put("aws.tags."+common.DeDot(*tag.Key), common.DeDot(*tag.Value))
			}
			events[autoScalingGroupName].RootFields.Put("aws.cloudwatch.namespace", cloudWatchNameSpaceAutoScaling)
			events[autoScalingGroupName].RootFields.Put("aws.dimensions."+dimensionName, autoScalingGroupName)
			events[autoScalingGroupName].RootFields.Put("aws.autoscaling.arn", autoScalingGroupName2Detail[autoScalingGroupName].AutoScalingGroupARN)
			events[autoScalingGroupName].RootFields.Put("aws.autoscaling.metrics."+metricName+"."+stat, output.Values[timestampIdx])

		}
	}

	return events, nil
}

func (m *MetricSet) generateMetricDataQueries(listMetricsOutput []cloudwatchTypes.Metric, period time.Duration, metricStats map[string][]string) []cloudwatchTypes.MetricDataQuery {
	var totalMetricDataQueries []cloudwatchTypes.MetricDataQuery
	for i, listMetric := range listMetricsOutput {
		metricDataQueries := m.generateMetricDataQuery(listMetric, i, period, metricStats)
		if len(metricDataQueries) == 0 {
			continue
		}
		totalMetricDataQueries = append(totalMetricDataQueries, metricDataQueries...)
	}
	return totalMetricDataQueries

}

func (m *MetricSet) generateMetricDataQuery(metric cloudwatchTypes.Metric, index int, period time.Duration, metricStats map[string][]string) []cloudwatchTypes.MetricDataQuery {
	var metricDataQueries []cloudwatchTypes.MetricDataQuery
	periodInSeconds := int32(period.Seconds())
	metricDims := metric.Dimensions
	for _, dim := range metricDims {
		metricName := *metric.MetricName
		stats, exists := metricStats[metricName]
		if !exists {
			continue
		}
		for _, s := range stats {
			id := metricsetName + metricName + s + strconv.Itoa(index)
			label := *dim.Name + labelSeparator + *dim.Value + labelSeparator + metricName + labelSeparator + s
			metricDataQueries = append(metricDataQueries, cloudwatchTypes.MetricDataQuery{
				Id: &id,
				MetricStat: &cloudwatchTypes.MetricStat{
					Period: &periodInSeconds,
					Stat:   &s,
					Metric: &metric,
				},
				Label: &label,
			})
		}
	}
	return metricDataQueries
}

func getAutoScalingGroupsPerRegion(svc *autoscaling.Client) ([]autoscalingTypes.AutoScalingGroup, error) {
	var autoScalingGroupsOutputs []autoscalingTypes.AutoScalingGroup
	p := autoscaling.NewDescribeAutoScalingGroupsPaginator(svc, &autoscaling.DescribeAutoScalingGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		autoScalingGroupsOutputs = append(autoScalingGroupsOutputs, page.AutoScalingGroups...)
	}
	return autoScalingGroupsOutputs, nil
}

func getAutoScalingGroupTags(autoScalingGroupsOutputs []autoscalingTypes.AutoScalingGroup) map[string][]resourcegroupstaggingapiTypes.Tag {
	tags := make(map[string][]resourcegroupstaggingapiTypes.Tag)
	for _, g := range autoScalingGroupsOutputs {
		for _, t := range g.Tags {
			tags[*g.AutoScalingGroupName] = append(tags[*g.AutoScalingGroupName], resourcegroupstaggingapiTypes.Tag{Key: t.Key, Value: t.Value})
		}
	}
	return tags
}

func convertConfigMetricStats(configs []MetricConfigs) map[string][]string {
	if configs == nil {
		return defaultMetricStats
	}
	metricStats := make(map[string][]string)
	for _, config := range configs {
		for _, metricName := range config.MetricName {
			metricStats[metricName] = append(metricStats[metricName], config.Statistic...)
		}
	}
	return metricStats

}
