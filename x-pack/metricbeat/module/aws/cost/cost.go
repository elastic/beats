// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cost

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer/costexploreriface"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"

	"context"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
)

var (
	metricsetName = "cost"
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
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}

	return &MetricSet{
		MetricSet: metricSet,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// Get startDate and endDate
	startDate, endDate := getStartDateEndDate(m.Period)

	awsConfig := m.MetricSet.AwsConfig.Copy()
	svc := costexplorer.New(awsConfig)
	timePeriod := costexplorer.DateInterval{
		Start: awssdk.String(startDate),
		End:   awssdk.String(endDate),
	}

	var events []mb.Event
	// Get total cost from GetCostAndUsage with group by type "TAG"
	eventsByTag := m.getCostGroupByTag(timePeriod, svc, startDate, endDate)
	events = append(events, eventsByTag...)

	// Get total cost from GetCostAndUsage with group by type "DIMENSION"
	eventsByAZ := m.getCostGroupByDimension(timePeriod, svc, startDate, endDate)
	events = append(events, eventsByAZ...)

	// report events
	for _, event := range events {
		if reported := report.Event(event); !reported {
			m.Logger().Debug("Fetch interrupted, failed to emit event")
			return nil
		}
	}
	return nil
}

func (m *MetricSet) getCostGroupByTag(timePeriod costexplorer.DateInterval, svc costexploreriface.ClientAPI, startDate string, endDate string) []mb.Event {
	var events []mb.Event
	groupByTagCostInput := costexplorer.GetCostAndUsageInput{
		Granularity: costexplorer.GranularityDaily,
		// no permission for "NetAmortizedCost" and "NetUnblendedCost"
		Metrics: []string{"AmortizedCost", "BlendedCost",
			"NormalizedUsageAmount", "UnblendedCost", "UsageQuantity"},
		TimePeriod: &timePeriod,
		GroupBy: []costexplorer.GroupDefinition{
			{
				Key:  awssdk.String("aws:createdBy"),
				Type: costexplorer.GroupDefinitionTypeTag,
			},
		},
	}

	groupByTagCostReq := svc.GetCostAndUsageRequest(&groupByTagCostInput)
	groupByTagOutput, err := groupByTagCostReq.Send(context.Background())
	if err != nil {
		err = fmt.Errorf("costexplorer GetCostAndUsageRequest failed: %w", err)
		m.Logger().Errorf(err.Error())
		return nil
	}

	if len(groupByTagOutput.ResultsByTime) > 0 {
		costResultGroups := groupByTagOutput.ResultsByTime[0].Groups
		for _, group := range costResultGroups {
			event := m.addCostMetrics(group.Metrics, groupByTagOutput.GroupDefinitions[0], startDate, endDate)
			for _, key := range group.Keys {
				tagKey, tagValue := parseGroupKey(key)
				event.MetricSetFields.Put("resourceTags."+tagKey, tagValue)
			}

			events = append(events, event)
		}
	}
	return events
}

func (m *MetricSet) getCostGroupByDimension(timePeriod costexplorer.DateInterval, svc costexploreriface.ClientAPI, startDate string, endDate string) []mb.Event {
	var events []mb.Event

	groupByAZCostInput := costexplorer.GetCostAndUsageInput{
		Granularity: costexplorer.GranularityDaily,
		// no permission for "NetAmortizedCost" and "NetUnblendedCost"
		Metrics: []string{"AmortizedCost", "BlendedCost",
			"NormalizedUsageAmount", "UnblendedCost", "UsageQuantity"},
		TimePeriod: &timePeriod,
		GroupBy: []costexplorer.GroupDefinition{
			{
				Key:  awssdk.String("AZ"),
				Type: costexplorer.GroupDefinitionTypeDimension,
			},
		},
	}

	groupByAZCostReq := svc.GetCostAndUsageRequest(&groupByAZCostInput)
	groupByAZOutput, err := groupByAZCostReq.Send(context.Background())
	if err != nil {
		err = fmt.Errorf("costexplorer GetCostAndUsageRequest failed: %w", err)
		m.Logger().Errorf(err.Error())
		return nil
	}

	if len(groupByAZOutput.ResultsByTime) > 0 {
		costResultGroups := groupByAZOutput.ResultsByTime[0].Groups
		for _, group := range costResultGroups {
			if group.Keys[0] == "NoAZ" {
				continue
			}

			event := m.addCostMetrics(group.Metrics, groupByAZOutput.GroupDefinitions[0], startDate, endDate)
			event.MetricSetFields.Put("availability_zone", group.Keys[0])
			events = append(events, event)
		}
	}
	return events
}

func (m *MetricSet) addCostMetrics(metrics map[string]costexplorer.MetricValue, groupDefinition costexplorer.GroupDefinition, startDate string, endDate string) mb.Event {
	event := aws.InitEvent("", m.AccountName, m.AccountID)

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

func getStartDateEndDate(period time.Duration) (startDate string, endDate string) {
	currentTime := time.Now()
	startTime := currentTime.Add(period * -1)
	startDate = startTime.Format("2006-01-02")
	endDate = currentTime.Format("2006-01-02")
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
