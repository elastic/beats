// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sysmetric

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
)

// collect function collects all the system metric information into an instance of collectededData
func (m *MetricSet) collect(ctx context.Context, collector sysmetricCollectMethod) (out *collectedData, err error) {
	out = &collectedData{}
	if out.sysmetricMetrics, err = collector.sysmetricMetric(ctx); err != nil {
		return nil, fmt.Errorf("error getting system metrics %w", err)
	}
	return out, nil
}

// collectAndTransform is called by the Fetch method, which is the one
// that "loads" the data into Elasticsearch.
func (m *MetricSet) collectAndTransform(ctx context.Context) ([]mb.Event, error) {
	collectedMetricsData, err := m.collect(ctx, m.collector)
	if err != nil {
		return nil, fmt.Errorf("error collecting data %w", err)
	}
	return m.transform(collectedMetricsData), nil
}

// Transform function Transforms the data to create a Kibana/Elasticsearch friendly JSON.
// Data from Oracle is pretty fragmented by design so a lot of data was necessary.
// Data is organized by sysmetric entity that contains metrics details.
func (m *MetricSet) transform(in *collectedData) []mb.Event {
	sysMetric := m.addSysmetricData(in.sysmetricMetrics)
	events := make([]mb.Event, 0)
	for _, v := range sysMetric {
		events = append(events, mb.Event{MetricSetFields: v})
	}
	return events
}
