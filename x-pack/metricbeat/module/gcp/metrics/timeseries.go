// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func getKeyValue(m mapstr.M, field string) string {
	val, err := m.GetValue(field)
	if err != nil {
		return ""
	}

	strVal, ok := val.(string)
	if !ok {
		return ""
	}

	return strVal
}

// createGroupingKey returns a key to group metrics by dimensions.
//
// At a high level, the key is made of the following components:
//   - @timestamp
//   - list dimension values
func createGroupingKey(kv KeyValuePoint) string {
	accountID := getKeyValue(kv.ECS, "cloud.account.id")
	az := getKeyValue(kv.ECS, "cloud.availability_zone")
	instanceID := getKeyValue(kv.ECS, "cloud.instance.id")
	provider := getKeyValue(kv.ECS, "cloud.provider")
	region := getKeyValue(kv.ECS, "cloud.region")

	dimensionsKey := fmt.Sprintf("%d_%s_%s_%s_%s_%s_%s",
		kv.Timestamp.UnixNano(),
		accountID,
		az,
		instanceID,
		provider,
		region,
		kv.Labels,
	)

	return dimensionsKey
}

// groupMetricsByDimensions returns a map of metrics grouped by dimensions.
func groupMetricsByDimensions(keyValues []KeyValuePoint) map[string][]KeyValuePoint {
	groupedMetrics := make(map[string][]KeyValuePoint)

	for _, kv := range keyValues {
		dimensionsKey := createGroupingKey(kv)
		groupedMetrics[dimensionsKey] = append(groupedMetrics[dimensionsKey], kv)
	}

	return groupedMetrics
}

// createEventsFromGroups returns a slice of events from the metric groups.
//
// Each group is made or one or more metrics, so the function collapses the
// metrics in each group into a single event:
//
//	[]KeyValuePoint -> mb.Event
//
// Collapsing the metrics in each group into a single event should not cause
// any loss of information, since all metrics in a group share the same timestamp
// and dimensions.
func createEventsFromGroups(service string, groups map[string][]KeyValuePoint) []mb.Event {
	events := make([]mb.Event, 0, len(groups))

	for _, group := range groups {
		event := mb.Event{
			Timestamp: group[0].Timestamp,
			ModuleFields: mapstr.M{
				"labels": group[0].Labels,
			},
			MetricSetFields: mapstr.M{},
		}

		for _, singleEvent := range group {
			// Add the metric values to the event.
			_, _ = event.MetricSetFields.Put(singleEvent.Key, singleEvent.Value)
		}

		if service == "compute" {
			event.RootFields = addHostFields(group)
		} else {
			event.RootFields = group[0].ECS
		}

		events = append(events, event)
	}

	return events
}

// groupTimeSeries groups TimeSeries into Elasticsearch friendly events.
//
// By grouping multiple TimeSeries (according to @timestamp and dimensions) into single event,
// we can avoid sending events with a single metric.
func (m *MetricSet) groupTimeSeries(ctx context.Context, timeSeries []timeSeriesWithAligner, defaultMetadataService gcp.MetadataService, mapper *incomingFieldMapper) map[string][]KeyValuePoint {
	metadataService := defaultMetadataService

	var kvs []KeyValuePoint

	for _, tsa := range timeSeries {
		aligner := tsa.aligner
		for _, ts := range tsa.timeSeries {
			if defaultMetadataService == nil {
				metadataService = gcp.NewStackdriverMetadataServiceForTimeSeries(ts)
			}
			sdCollectorInputData := gcp.NewStackdriverCollectorInputData(ts, m.config.ProjectID, m.config.Zone, m.config.Region, m.config.Regions)
			keyValues := mapper.mapTimeSeriesToKeyValuesPoints(ts, aligner)

			for i := range keyValues {
				sdCollectorInputData.Timestamp = &keyValues[i].Timestamp

				metadataCollectorData, err := metadataService.Metadata(ctx, sdCollectorInputData.TimeSeries)
				if err != nil {
					m.Logger().Error("error trying to retrieve labels from metric event")
					continue
				}

				keyValues[i].ECS = metadataCollectorData.ECS
				keyValues[i].Labels = metadataCollectorData.Labels
			}

			kvs = append(kvs, keyValues...)
		}
	}

	// Group the data by common fields (dimensions)
	groupedMetrics := groupMetricsByDimensions(kvs)

	return groupedMetrics
}
