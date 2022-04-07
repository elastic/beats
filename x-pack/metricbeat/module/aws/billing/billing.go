// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/costexploreriface"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/organizationsiface"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/metricbeat/mb"
	awscommon "github.com/elastic/beats/v8/x-pack/libbeat/common/aws"
	"github.com/elastic/beats/v8/x-pack/metricbeat/module/aws"
)

var (
	metricsetName  = "billing"
	regionName     = "us-east-1"
	labelSeparator = "|"

	// This list is from https://github.com/aws/aws-sdk-go-v2/blob/master/service/costexplorer/api_enums.go#L60-L90
	supportedDimensionKeys = []string{
		"AZ", "INSTANCE_TYPE", "LINKED_ACCOUNT", "OPERATION", "PURCHASE_TYPE",
		"REGION", "SERVICE", "USAGE_TYPE", "USAGE_TYPE_GROUP", "RECORD_TYPE",
		"OPERATING_SYSTEM", "TENANCY", "SCOPE", "PLATFORM", "SUBSCRIPTION_ID",
		"LEGAL_ENTITY_NAME", "DEPLOYMENT_OPTION", "DATABASE_ENGINE",
		"CACHE_ENGINE", "INSTANCE_TYPE_FAMILY", "BILLING_ENTITY",
		"RESERVATION_ID",
	}

	dateLayout = "2006-01-02"
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
	logger             *logp.Logger
	CostExplorerConfig CostExplorerConfig `config:"cost_explorer_config"`
}

// Config holds a configuration specific for billing metricset.
type CostExplorerConfig struct {
	GroupByDimensionKeys []string `config:"group_by_dimension_keys"`
	GroupByTagKeys       []string `config:"group_by_tag_keys"`
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
		CostExplorerConfig CostExplorerConfig `config:"cost_explorer_config"`
	}{}

	err = base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, fmt.Errorf("error unpack raw module config using UnpackConfig: %w", err)
	}

	logger.Debugf("cost explorer config = %s", config)

	return &MetricSet{
		MetricSet:          metricSet,
		logger:             logger,
		CostExplorerConfig: config.CostExplorerConfig,
	}, nil
}

// Validate checks if given dimension keys are supported.
func (c CostExplorerConfig) Validate() error {
	for _, key := range c.GroupByDimensionKeys {
		supported, _ := aws.StringInSlice(key, supportedDimensionKeys)
		if !supported {
			return fmt.Errorf("costexplorer GetCostAndUsageRequest does not support dimension key: %s", key)
		}
	}
	return nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	var config aws.Config
	err := m.Module().UnpackConfig(&config)
	if err != nil {
		return nil
	}
	monitoringServiceName := awscommon.CreateServiceName("monitoring", config.AWSConfig.FIPSEnabled, regionName)
	// Get startDate and endDate
	startDate, endDate := getStartDateEndDate(m.Period)

	// Get startTime and endTime
	startTime, endTime := aws.GetStartTimeEndTime(m.Period, m.Latency)

	// get cost metrics from cost explorer
	awsConfig := m.MetricSet.AwsConfig.Copy()
	svcCostExplorer := costexplorer.New(awscommon.EnrichAWSConfigWithEndpoint(
		m.Endpoint, monitoringServiceName, "", awsConfig))

	awsConfig.Region = regionName
	svcCloudwatch := cloudwatch.New(awscommon.EnrichAWSConfigWithEndpoint(
		m.Endpoint, monitoringServiceName, regionName, awsConfig))

	timePeriod := costexplorer.DateInterval{
		Start: awssdk.String(startDate),
		End:   awssdk.String(endDate),
	}

	var events []mb.Event

	// Get estimated charges from CloudWatch
	eventsCW := m.getCloudWatchBillingMetrics(svcCloudwatch, startTime, endTime)
	events = append(events, eventsCW...)

	// Get total cost from Cost Explorer GetCostAndUsage with group by type "DIMENSION" and "TAG"
	eventsCE := m.getCostGroupBy(svcCostExplorer, m.CostExplorerConfig.GroupByDimensionKeys, m.CostExplorerConfig.GroupByTagKeys, timePeriod, startDate, endDate)
	events = append(events, eventsCE...)

	// report events
	for _, event := range events {
		if reported := report.Event(event); !reported {
			m.Logger().Debug("Fetch interrupted, failed to emit event")
			return nil
		}
	}
	return nil
}

func (m *MetricSet) getCloudWatchBillingMetrics(
	svcCloudwatch cloudwatchiface.ClientAPI,
	startTime time.Time,
	endTime time.Time) []mb.Event {
	var events []mb.Event
	namespace := "AWS/Billing"
	listMetricsOutput, err := aws.GetListMetricsOutput(namespace, regionName, svcCloudwatch)
	if err != nil {
		m.Logger().Error(err.Error())
		return nil
	}

	if listMetricsOutput == nil || len(listMetricsOutput) == 0 {
		return events
	}

	metricDataQueriesTotal := constructMetricQueries(listMetricsOutput, m.Period)
	metricDataOutput, err := aws.GetMetricDataResults(metricDataQueriesTotal, svcCloudwatch, startTime, endTime)
	if err != nil {
		err = fmt.Errorf("aws GetMetricDataResults failed with %w, skipping region %s", err, regionName)
		m.Logger().Error(err.Error())
		return nil
	}

	// Find a timestamp for all metrics in output
	timestamp := aws.FindTimestamp(metricDataOutput)
	if timestamp.IsZero() {
		return nil
	}

	for _, output := range metricDataOutput {
		if len(output.Values) == 0 {
			continue
		}
		exists, timestampIdx := aws.CheckTimestampInArray(timestamp, output.Timestamps)
		if exists {
			labels := strings.Split(*output.Label, labelSeparator)

			event := aws.InitEvent("", m.AccountName, m.AccountID, timestamp)
			event.MetricSetFields.Put(labels[0], output.Values[timestampIdx])

			i := 1
			for i < len(labels)-1 {
				event.MetricSetFields.Put(labels[i], labels[i+1])
				i += 2
			}
			event.Timestamp = endTime
			events = append(events, event)
		}
	}
	return events
}

func (m *MetricSet) getCostGroupBy(svcCostExplorer costexploreriface.ClientAPI, groupByDimKeys []string, groupByTags []string, timePeriod costexplorer.DateInterval, startDate string, endDate string) []mb.Event {
	var events []mb.Event

	// get linked account IDs and names
	accounts := map[string]string{}
	var config aws.Config
	err := m.Module().UnpackConfig(&config)
	if err != nil {
		return nil
	}
	if ok, _ := aws.StringInSlice("LINKED_ACCOUNT", groupByDimKeys); ok {
		awsConfig := m.MetricSet.AwsConfig.Copy()
		organizationsServiceName := awscommon.CreateServiceName("organizations", config.AWSConfig.FIPSEnabled, regionName)

		svcOrg := organizations.New(awscommon.EnrichAWSConfigWithEndpoint(
			m.Endpoint, organizationsServiceName, regionName, awsConfig))
		accounts = m.getAccountName(svcOrg)
	}

	groupBys := getGroupBys(groupByTags, groupByDimKeys)
	for _, groupBy := range groupBys {
		var groupDefs []costexplorer.GroupDefinition

		if groupBy.dimension != "" {
			groupDefs = append(groupDefs, costexplorer.GroupDefinition{
				Key:  awssdk.String(groupBy.dimension),
				Type: costexplorer.GroupDefinitionTypeDimension,
			})
		}

		if groupBy.tag != "" {
			groupDefs = append(groupDefs, costexplorer.GroupDefinition{
				Key:  awssdk.String(groupBy.tag),
				Type: costexplorer.GroupDefinitionTypeTag,
			})
		}

		groupByCostInput := costexplorer.GetCostAndUsageInput{
			Granularity: costexplorer.GranularityDaily,
			// no permission for "NetAmortizedCost" and "NetUnblendedCost"
			Metrics: []string{"AmortizedCost", "BlendedCost",
				"NormalizedUsageAmount", "UnblendedCost", "UsageQuantity"},
			TimePeriod: &timePeriod,
			// Only two values for GroupBy are allowed
			GroupBy: groupDefs,
		}

		groupByCostReq := svcCostExplorer.GetCostAndUsageRequest(&groupByCostInput)
		groupByOutput, err := groupByCostReq.Send(context.Background())
		if err != nil {
			err = fmt.Errorf("costexplorer GetCostAndUsageRequest failed: %w", err)
			m.Logger().Errorf(err.Error())
			return nil
		}

		if len(groupByOutput.ResultsByTime) > 0 {
			costResultGroups := groupByOutput.ResultsByTime[0].Groups
			for _, group := range costResultGroups {
				event := m.addCostMetrics(group.Metrics, groupByOutput.GroupDefinitions[0], startDate, endDate)

				// generate unique event ID for each event
				eventID := startDate + endDate + *groupByOutput.GroupDefinitions[0].Key + string(groupByOutput.GroupDefinitions[0].Type)
				for _, key := range group.Keys {
					eventID += key
					// key value like db.t2.micro or Amazon Simple Queue Service belongs to dimension
					if !strings.Contains(key, "$") {
						event.MetricSetFields.Put("group_by."+groupBy.dimension, key)
						if groupBy.dimension == "LINKED_ACCOUNT" {
							if name, ok := accounts[key]; ok {
								event.RootFields.Put("aws.linked_account.id", key)
								event.RootFields.Put("aws.linked_account.name", name)
							}
						}
						continue
					}

					// tag key value is separated by $
					tagKey, tagValue := parseGroupKey(key)
					if tagValue != "" {
						event.MetricSetFields.Put("group_by."+tagKey, tagValue)
					}
				}

				t, err := time.Parse(dateLayout, endDate)
				if err == nil {
					event.Timestamp = t
				}

				event.ID = generateEventID(eventID)
				events = append(events, event)
			}
		}
	}
	return events
}

func (m *MetricSet) addCostMetrics(metrics map[string]costexplorer.MetricValue, groupDefinition costexplorer.GroupDefinition, startDate string, endDate string) mb.Event {
	event := aws.InitEvent("", m.AccountName, m.AccountID, time.Now())

	// add group definition
	event.MetricSetFields.Put("group_definition", common.MapStr{
		"key":  *groupDefinition.Key,
		"type": groupDefinition.Type,
	})

	for metricName, metricValues := range metrics {
		cost := metricValues
		costFloat, err := strconv.ParseFloat(*cost.Amount, 64)
		if err != nil {
			err = fmt.Errorf("strconv ParseFloat failed: %w", err)
			m.Logger().Errorf(err.Error())
			continue
		}

		value := common.MapStr{
			"amount": costFloat,
			"unit":   &cost.Unit,
		}

		event.MetricSetFields.Put(metricName, value)
		event.MetricSetFields.Put("start_date", startDate)
		event.MetricSetFields.Put("end_date", endDate)
	}
	return event
}

func constructMetricQueries(listMetricsOutput []cloudwatch.Metric, period time.Duration) []cloudwatch.MetricDataQuery {
	var metricDataQueries []cloudwatch.MetricDataQuery
	metricDataQueryEmpty := cloudwatch.MetricDataQuery{}
	for i, listMetric := range listMetricsOutput {
		metricDataQuery := createMetricDataQuery(listMetric, i, period)
		if metricDataQuery == metricDataQueryEmpty {
			continue
		}
		metricDataQueries = append(metricDataQueries, metricDataQuery)
	}
	return metricDataQueries
}

func createMetricDataQuery(metric cloudwatch.Metric, index int, period time.Duration) (metricDataQuery cloudwatch.MetricDataQuery) {
	statistic := "Maximum"
	periodInSeconds := int64(period.Seconds())
	id := metricsetName + strconv.Itoa(index)
	metricDims := metric.Dimensions
	metricName := *metric.MetricName

	label := metricName + labelSeparator
	for _, dim := range metricDims {
		label += *dim.Name + labelSeparator + *dim.Value + labelSeparator
	}

	metricDataQuery = cloudwatch.MetricDataQuery{
		Id: &id,
		MetricStat: &cloudwatch.MetricStat{
			Period: &periodInSeconds,
			Stat:   &statistic,
			Metric: &metric,
		},
		Label: &label,
	}
	return
}

func getStartDateEndDate(period time.Duration) (startDate string, endDate string) {
	currentTime := time.Now()
	startTime := currentTime.Add(period * -1)
	startDate = startTime.Format(dateLayout)
	endDate = currentTime.Format(dateLayout)
	return
}

func parseGroupKey(groupKey string) (tagKey string, tagValue string) {
	keys := strings.Split(groupKey, "$")
	if len(keys) == 2 {
		tagKey = keys[0]
		tagValue = keys[1]
	} else if len(keys) > 2 {
		tagKey = keys[0]
		tagValue = keys[1]
		for i := 2; i < len(keys); i++ {
			tagValue = tagValue + "$" + keys[i]
		}
	} else {
		tagKey = keys[0]
		tagValue = ""
	}
	return
}

type groupBy struct {
	tag       string
	dimension string
}

func getGroupBys(groupByTags []string, groupByDimKeys []string) []groupBy {
	var groupBys []groupBy

	if len(groupByTags) == 0 {
		groupByTags = []string{""}
	}
	if len(groupByDimKeys) == 0 {
		groupByDimKeys = []string{""}
	}

	for _, tagKey := range groupByTags {
		for _, dimKey := range groupByDimKeys {
			groupBy := groupBy{
				tag:       tagKey,
				dimension: dimKey,
			}
			groupBys = append(groupBys, groupBy)
		}
	}
	return groupBys
}

func generateEventID(eventID string) string {
	// create eventID using hash of startDate + endDate + groupDefinitionKey + groupDefinitionType + values
	// This will prevent more than one billing metric getting collected in the same day.
	h := sha256.New()
	h.Write([]byte(eventID))
	prefix := hex.EncodeToString(h.Sum(nil))
	return prefix[:20]
}

func (m *MetricSet) getAccountName(svc organizationsiface.ClientAPI) map[string]string {
	// construct ListAccountsInput
	ListAccountsInput := &organizations.ListAccountsInput{}
	req := svc.ListAccountsRequest(ListAccountsInput)
	p := organizations.NewListAccountsPaginator(req)

	accounts := map[string]string{}
	for p.Next(context.TODO()) {
		page := p.CurrentPage()
		for _, a := range page.Accounts {
			accounts[*a.Id] = *a.Name
		}
	}

	if err := p.Err(); err != nil {
		m.logger.Warnf("failed ListAccountsRequest", err)
	}
	return accounts
}
