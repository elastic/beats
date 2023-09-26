// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import (
	"context"
	"fmt"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
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

func createDimensionsKey(kv KeyValuePoint) string {
	// Figure the list of dimensions that we want to group by from kv.ECS and kv.Labels

	accountID := getKeyValue(kv.ECS, "cloud.account.id")
	az := getKeyValue(kv.ECS, "cloud.availability_zone")
	instanceID := getKeyValue(kv.ECS, "cloud.instance.id")
	provider := getKeyValue(kv.ECS, "cloud.provider")
	region := getKeyValue(kv.ECS, "cloud.region")

	dimensionsKey := fmt.Sprintf("%s_%s_%s_%s_%s_%s",
		accountID,
		az,
		instanceID,
		provider,
		region,
		kv.Labels,
	)

	return dimensionsKey
}

func groupMetricsByDimensions(keyValues []KeyValuePoint) map[string][]KeyValuePoint {
	groupedMetrics := make(map[string][]KeyValuePoint)

	for _, kv := range keyValues {
		dimensionsKey := createDimensionsKey(kv)
		groupedMetrics[dimensionsKey] = append(groupedMetrics[dimensionsKey], kv)
	}

	return groupedMetrics
}

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

// timeSeriesGrouped groups TimeSeries responses into common Elasticsearch friendly events. This is to avoid sending
// events with a single metric that shares info (like timestamp) with another event with a single metric too
func (m *MetricSet) timeSeriesGrouped(ctx context.Context, gcpService gcp.MetadataService, tsas []timeSeriesWithAligner, e *incomingFieldExtractor) map[string][]KeyValuePoint {
	metadataService := gcpService

	var kvs []KeyValuePoint

	for _, tsa := range tsas {
		aligner := tsa.aligner
		for _, ts := range tsa.timeSeries {
			keyValues := e.extractTimeSeriesMetricValues(ts, aligner)

			sdCollectorInputData := gcp.NewStackdriverCollectorInputData(ts, m.config.ProjectID, m.config.Zone, m.config.Region, m.config.Regions)
			if gcpService == nil {
				metadataService = gcp.NewStackdriverMetadataServiceForTimeSeries(ts)
			}

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
