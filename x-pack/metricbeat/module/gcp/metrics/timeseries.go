// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
)

//timeSeriesGrouped groups TimeSeries responses into common Elasticsearch friendly events. This is to avoid sending
// events with a single metric that shares info (like timestamp) with another event with a single metric too
func (m *MetricSet) timeSeriesGrouped(ctx context.Context, gcpService gcp.MetadataService, tsas []timeSeriesWithAligner, e *incomingFieldExtractor) map[string][]KeyValuePoint {
	eventGroups := make(map[string][]KeyValuePoint)

	metadataService := gcpService

	for _, tsa := range tsas {
		aligner := tsa.aligner
		for _, ts := range tsa.timeSeries {
			keyValues := e.extractTimeSeriesMetricValues(ts, aligner)

			sdCollectorInputData := gcp.NewStackdriverCollectorInputData(ts, m.config.ProjectID, m.config.Zone, m.config.Region)
			if gcpService == nil {
				metadataService = gcp.NewStackdriverMetadataServiceForTimeSeries(ts)
			}

			for i := range keyValues {
				sdCollectorInputData.Timestamp = &keyValues[i].Timestamp

				id, err := metadataService.ID(ctx, sdCollectorInputData)
				if err != nil {
					m.Logger().Errorf("error trying to retrieve ID from metric event '%v'", err)
					continue
				}

				metadataCollectorData, err := metadataService.Metadata(ctx, sdCollectorInputData.TimeSeries)
				if err != nil {
					m.Logger().Error("error trying to retrieve labels from metric event")
					continue
				}

				if _, ok := eventGroups[id]; !ok {
					eventGroups[id] = make([]KeyValuePoint, 0)
				}

				keyValues[i].ECS = metadataCollectorData.ECS
				keyValues[i].Labels = metadataCollectorData.Labels

				// Group the data into common events
				eventGroups[id] = append(eventGroups[id], keyValues[i])
			}
		}
	}

	return eventGroups
}
