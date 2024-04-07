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

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	costexplorertypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/aws/aws-sdk-go-v2/service/organizations"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	supportedDimensionKeys = costexplorertypes.Dimension("").Values()

	// this module doesn't currently support `COST_CATEGORY`, so we can use the default aws sdk list
	// values from https://github.com/aws/aws-sdk-go-v2/blob/main/service/costexplorer/types/enums.go#L455-L470
	supportedGroupByTypes = []costexplorertypes.GroupDefinitionType{
		costexplorertypes.GroupDefinitionTypeDimension,
		costexplorertypes.GroupDefinitionTypeTag,
	}
)

const (
	metricsetName = "billing"
	regionName    = "us-east-1"
	dateLayout    = "2006-01-02"

	defaultGroupByPrimaryType   = costexplorertypes.GroupDefinitionTypeDimension
	defaultGroupBySecondaryType = costexplorertypes.GroupDefinitionTypeTag
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

// CostExplorerConfig holds a configuration specific for billing metricset.
type CostExplorerConfig struct {
	GroupByPrimaryKeys   []string                              `config:"group_by_dimension_keys" config:"group_by_primary_keys"`
	GroupByPrimaryType   costexplorertypes.GroupDefinitionType `config:"group_by_primary_type"`
	GroupBySecondaryKeys []string                              `config:"group_by_tag_keys" config:"group_by_secondary_keys"`
	GroupBySecondaryType costexplorertypes.GroupDefinitionType `config:"group_by_secondary_type"`
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

	if config.CostExplorerConfig.GroupByPrimaryType == "" {
		config.CostExplorerConfig.GroupByPrimaryType = defaultGroupByPrimaryType
	}

	if config.CostExplorerConfig.GroupBySecondaryType == "" {
		config.CostExplorerConfig.GroupBySecondaryType = defaultGroupBySecondaryType
	}

	return &MetricSet{
		MetricSet:          metricSet,
		logger:             logger,
		CostExplorerConfig: config.CostExplorerConfig,
	}, nil
}

// Validate checks if given config is supported.
func (c CostExplorerConfig) Validate() error {
	// validate primary group by type
	if supported, err := validateGroupByType(c.GroupByPrimaryType); !supported {
		return err
	}

	// validate secondary group by type
	if supported, err := validateGroupByType(c.GroupBySecondaryType); !supported {
		return err
	}

	// validate dimension keys for primary if group by type is dimension
	if c.GroupByPrimaryType == costexplorertypes.GroupDefinitionTypeDimension {
		for _, key := range c.GroupByPrimaryKeys {
			supported, err := validateDimensionKey(key)
			if !supported {
				return err
			}
		}
	}

	// validate dimension keys for secondary if group by type is dimension
	if c.GroupBySecondaryType == costexplorertypes.GroupDefinitionTypeDimension {
		for _, key := range c.GroupBySecondaryKeys {
			supported, err := validateDimensionKey(key)
			if !supported {
				return err
			}
		}
	}

	return nil
}

// validateGroupByType checks if a string value for group by type is a supported value.
func validateGroupByType(groupByType costexplorertypes.GroupDefinitionType) (bool, error) {
	// handle setting a default value in New() call
	if string(groupByType) == "" {
		return true, nil
	}
	for _, key := range supportedGroupByTypes {
		if groupByType == key {
			return true, nil
		}
	}

	return false, fmt.Errorf("costexplorer GetCostAndUsageRequest or metricbeat module does not support group by type: %s", groupByType)
}

// validateDimensionKey checks if a string value for dimension key is a supported value.
func validateDimensionKey(dimensionKey string) (bool, error) {
	for _, key := range supportedDimensionKeys {
		if costexplorertypes.Dimension(dimensionKey) == key {
			return true, nil
		}
	}
	return false, fmt.Errorf("costexplorer GetCostAndUsageRequest does not support dimension key: %s", dimensionKey)
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	var config aws.Config
	err := m.Module().UnpackConfig(&config)
	if err != nil {
		return err
	}
	// Get startDate and endDate
	startDate, endDate := getStartDateEndDate(m.Period)

	// Get startTime and endTime
	startTime, endTime := aws.GetStartTimeEndTime(time.Now(), m.Period, m.Latency)

	// get cost metrics from cost explorer
	awsBeatsConfig := m.MetricSet.AwsConfig.Copy()
	svcCostExplorer := costexplorer.NewFromConfig(awsBeatsConfig, func(o *costexplorer.Options) {
		if config.AWSConfig.FIPSEnabled {
			o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
		}

	})

	awsBeatsConfig.Region = regionName
	svcCloudwatch := cloudwatch.NewFromConfig(awsBeatsConfig, func(o *cloudwatch.Options) {
		if config.AWSConfig.FIPSEnabled {
			o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
		}
	})

	timePeriod := costexplorertypes.DateInterval{
		Start: awssdk.String(startDate),
		End:   awssdk.String(endDate),
	}

	var events []mb.Event

	// Get estimated charges from CloudWatch
	eventsCW := m.getCloudWatchBillingMetrics(svcCloudwatch, startTime, endTime)
	events = append(events, eventsCW...)

	// Get total cost from Cost Explorer GetCostAndUsage with group by type "DIMENSION" and "TAG"
	eventsCE := m.getCostGroupBy(svcCostExplorer, m.CostExplorerConfig.GroupByPrimaryType, m.CostExplorerConfig.GroupByPrimaryKeys, m.CostExplorerConfig.GroupBySecondaryType, m.CostExplorerConfig.GroupBySecondaryKeys, timePeriod, startDate, endDate)
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
	svcCloudwatch *cloudwatch.Client,
	startTime time.Time,
	endTime time.Time) []mb.Event {
	var events []mb.Event
	namespace := "AWS/Billing"
	listMetricsOutput, err := aws.GetListMetricsOutput(namespace, regionName, m.Period, m.IncludeLinkedAccounts, m.MonitoringAccountID, svcCloudwatch)
	if err != nil {
		m.Logger().Error(err.Error())
		return nil
	}

	if len(listMetricsOutput) == 0 {
		return events
	}

	metricDataQueriesTotal := constructMetricQueries(listMetricsOutput, m.DataGranularity)
	metricDataOutput, err := aws.GetMetricDataResults(metricDataQueriesTotal, svcCloudwatch, startTime, endTime)
	if err != nil {
		err = fmt.Errorf("aws GetMetricDataResults failed with %w, skipping region %s", err, regionName)
		m.Logger().Error(err.Error())
		return nil
	}

	for _, output := range metricDataOutput {
		if len(output.Values) == 0 {
			continue
		}
		for valI, metricDataResultValue := range output.Values {
			labels := strings.Split(*output.Label, aws.LabelConst.LabelSeparator)
			event := mb.Event{}
			if labels[aws.LabelConst.AccountIdIdx] != "" {
				event = aws.InitEvent("", labels[aws.LabelConst.AccountLabelIdx], labels[aws.LabelConst.AccountIdIdx], output.Timestamps[valI], "")
			} else {
				event = aws.InitEvent("", m.MonitoringAccountName, m.MonitoringAccountID, output.Timestamps[valI], "")
			}

			_, _ = event.MetricSetFields.Put(labels[aws.LabelConst.MetricNameIdx], metricDataResultValue)

			i := aws.LabelConst.BillingDimensionStartIdx
			for i < len(labels)-1 {
				_, _ = event.MetricSetFields.Put(labels[i], labels[i+1])
				i += 2
			}
			event.Timestamp = endTime
			events = append(events, event)
		}
	}
	return events
}

func (m *MetricSet) getCostGroupBy(svcCostExplorer *costexplorer.Client, groupByPrimaryType costexplorertypes.GroupDefinitionType, groupByPrimaryKeys []string, groupBySecondaryType costexplorertypes.GroupDefinitionType, groupBySecondaryKeys []string, timePeriod costexplorertypes.DateInterval, startDate string, endDate string) []mb.Event {
	var events []mb.Event

	// get linked account IDs and names
	accounts := map[string]string{}
	var config aws.Config
	err := m.Module().UnpackConfig(&config)
	if err != nil {
		return nil
	}

	if ok, _ := aws.StringInSlice("LINKED_ACCOUNT", append(groupByPrimaryKeys, groupBySecondaryKeys...)); ok {
		awsConfig := m.MetricSet.AwsConfig.Copy()

		svcOrg := organizations.NewFromConfig(awsConfig, func(o *organizations.Options) {
			if config.AWSConfig.FIPSEnabled {
				o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
			}
		})
		accounts = m.getAccountName(svcOrg)
	}

	groupBys := getGroupBys(groupBySecondaryKeys, groupByPrimaryKeys)
	for _, groupBy := range groupBys {
		var groupDefs []costexplorertypes.GroupDefinition

		if groupBy.primary != "" {
			groupDefs = append(groupDefs, costexplorertypes.GroupDefinition{
				Key:  awssdk.String(groupBy.primary),
				Type: groupByPrimaryType,
			})
		}

		if groupBy.secondary != "" {
			groupDefs = append(groupDefs, costexplorertypes.GroupDefinition{
				Key:  awssdk.String(groupBy.secondary),
				Type: groupBySecondaryType,
			})
		}

		groupByCostInput := costexplorer.GetCostAndUsageInput{
			Granularity: costexplorertypes.GranularityDaily,
			// no permission for "NetAmortizedCost" and "NetUnblendedCost"
			Metrics: []string{"AmortizedCost", "BlendedCost",
				"NormalizedUsageAmount", "UnblendedCost", "UsageQuantity"},
			TimePeriod: &timePeriod,
			// Only two values for GroupBy are allowed
			GroupBy: groupDefs,
		}

		groupByOutput, err := svcCostExplorer.GetCostAndUsage(context.Background(), &groupByCostInput)
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

					// primary linked account dimension processing
					if groupByPrimaryType == costexplorertypes.GroupDefinitionTypeDimension && !strings.Contains(key, "$") {
						_, _ = event.MetricSetFields.Put("group_by."+groupBy.primary, key)
						if groupBy.primary == "LINKED_ACCOUNT" {
							if name, ok := accounts[key]; ok {
								_, _ = event.RootFields.Put("aws.linked_account.id", key)
								_, _ = event.RootFields.Put("aws.linked_account.name", name)
							}
						}
						continue
					}

					// secondary linked account dimension processing
					if groupBySecondaryType == costexplorertypes.GroupDefinitionTypeDimension && !strings.Contains(key, "$") {
						_, _ = event.MetricSetFields.Put("group_by."+groupBy.secondary, key)
						if groupBy.secondary == "LINKED_ACCOUNT" {
							if name, ok := accounts[key]; ok {
								_, _ = event.RootFields.Put("aws.linked_account.id", key)
								_, _ = event.RootFields.Put("aws.linked_account.name", name)
							}
						}
						continue
					}

					if groupByPrimaryType == costexplorertypes.GroupDefinitionTypeTag {
						// tag key value is separated by $
						tagKey, tagValue := parseGroupKey(key)
						if tagValue != "" {
							_, _ = event.MetricSetFields.Put("group_by."+tagKey, tagValue)
						}
					}

					if groupBySecondaryType == costexplorertypes.GroupDefinitionTypeTag {
						// tag key value is separated by $
						tagKey, tagValue := parseGroupKey(key)
						if tagValue != "" {
							_, _ = event.MetricSetFields.Put("group_by."+tagKey, tagValue)
						}
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

func (m *MetricSet) addCostMetrics(metrics map[string]costexplorertypes.MetricValue, groupDefinition costexplorertypes.GroupDefinition, startDate string, endDate string) mb.Event {
	event := aws.InitEvent("", m.MonitoringAccountName, m.MonitoringAccountID, time.Now(), "")

	// add group definition
	_, _ = event.MetricSetFields.Put("group_definition", mapstr.M{
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

		value := mapstr.M{
			"amount": costFloat,
			"unit":   &cost.Unit,
		}

		_, _ = event.MetricSetFields.Put(metricName, value)
		_, _ = event.MetricSetFields.Put("start_date", startDate)
		_, _ = event.MetricSetFields.Put("end_date", endDate)
	}
	return event
}

func constructMetricQueries(listMetricsOutput []aws.MetricWithID, dataGranularity time.Duration) []types.MetricDataQuery {
	var metricDataQueries []types.MetricDataQuery
	metricDataQueryEmpty := types.MetricDataQuery{}
	for i, listMetric := range listMetricsOutput {
		metricDataQuery := createMetricDataQuery(listMetric, i, dataGranularity)
		if metricDataQuery == metricDataQueryEmpty {
			continue
		}
		metricDataQueries = append(metricDataQueries, metricDataQuery)
	}
	return metricDataQueries
}

func createMetricDataQuery(metric aws.MetricWithID, index int, dataGranularity time.Duration) types.MetricDataQuery {
	statistic := "Maximum"
	dataGranularityInSeconds := int32(dataGranularity.Seconds())
	id := metricsetName + strconv.Itoa(index)
	metricDims := metric.Metric.Dimensions
	metricName := *metric.Metric.MetricName

	label := strings.Join([]string{metric.AccountID, aws.LabelConst.AccountLabel, metricName}, aws.LabelConst.LabelSeparator)
	for _, dim := range metricDims {
		label += aws.LabelConst.LabelSeparator + *dim.Name + aws.LabelConst.LabelSeparator + *dim.Value
	}

	metricDataQuery := types.MetricDataQuery{
		Id: &id,
		MetricStat: &types.MetricStat{
			Period: &dataGranularityInSeconds,
			Stat:   &statistic,
			Metric: &metric.Metric,
		},
		Label: &label,
	}

	if metric.AccountID != "" {
		metricDataQuery.AccountId = &metric.AccountID
	}
	return metricDataQuery
}

func getStartDateEndDate(period time.Duration) (string, string) {
	currentTime := time.Now()
	startTime := currentTime.Add(period * -1)
	startDate := startTime.Format(dateLayout)
	endDate := currentTime.Format(dateLayout)
	return startDate, endDate
}

func parseGroupKey(groupKey string) (string, string) {
	var tagKey, tagValue string
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
	}

	return tagKey, tagValue
}

type groupBy struct {
	secondary string
	primary   string
}

func getGroupBys(groupBySecondaryKeys []string, groupByPrimaryKeys []string) []groupBy {
	var groupBys []groupBy

	if len(groupBySecondaryKeys) == 0 {
		groupBySecondaryKeys = []string{""}
	}
	if len(groupByPrimaryKeys) == 0 {
		groupByPrimaryKeys = []string{""}
	}

	for _, secondaryKey := range groupBySecondaryKeys {
		for _, primaryKey := range groupByPrimaryKeys {
			groupBy := groupBy{
				secondary: secondaryKey,
				primary:   primaryKey,
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

func (m *MetricSet) getAccountName(svc *organizations.Client) map[string]string {
	// construct ListAccountsInput
	listAccountsInput := &organizations.ListAccountsInput{}
	paginator := organizations.NewListAccountsPaginator(svc, listAccountsInput)

	accounts := map[string]string{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			m.Logger().Warnf("an error occurred while listing account: %s", err.Error())
			return accounts
		}
		for _, a := range page.Accounts {
			accounts[*a.Id] = *a.Name
		}
	}

	return accounts
}
