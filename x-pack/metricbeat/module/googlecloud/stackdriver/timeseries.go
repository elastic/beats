// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stackdriver

import (
	"context"

	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"

	"github.com/elastic/beats/x-pack/metricbeat/module/googlecloud"
)

//timeSeriesGrouped groups TimeSeries responses into common Elasticsearch friendly events. This is to avoid sending
// events with a single metric that shares info (like timestamp) with another event with a single metric too
func (m *MetricSet) timeSeriesGrouped(ctx context.Context, gcpService googlecloud.MetadataService, tss []*monitoringpb.TimeSeries, e *incomingFieldExtractor) (map[string][]KeyValuePoint, error) {
	eventGroups := make(map[string][]KeyValuePoint)

	for _, ts := range tss {
		if gcpService == nil {
			gcpService = googlecloud.NewStackdriverMetadataServiceForTimeSeries(ts)
		}

		keyValues, err := e.extractTimeSeriesMetricValues(ts)
		if err != nil {
			return nil, err
		}

		sdCollectorInputData := googlecloud.NewStackdriverCollectorInputData(ts, m.config.ProjectID, m.config.Zone, m.config.Region)

		for i := range keyValues {
			sdCollectorInputData.Timestamp = &keyValues[i].Timestamp

			id, err := gcpService.ID(ctx, sdCollectorInputData)
			if err != nil {
				m.Logger().Errorf("error trying to retrieve ID from metric event '%v'", err)
				continue
			}

			metadataCollectorData, err := gcpService.Metadata(ctx, sdCollectorInputData.TimeSeries)
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

	return eventGroups, nil
}
