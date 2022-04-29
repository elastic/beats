// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sysmetric

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
)

// extract is the E of a ETL processing. Gets all the system metric information into an instance of extractedData
func (m *MetricSet) extract(ctx context.Context, extractor sysmetricExtractMethod) (out *extractedData, err error) {
	out = &extractedData{}
	if out.sysmetricMetrics, err = extractor.sysmetricMetric(ctx); err != nil {
		return nil, fmt.Errorf("error getting system metrics %w", err)
	}
	return out, nil
}

// extractAndTransform just composes the ET operations from a ETL. It's called by the Fetch method, which is the one
// that "loads" the data into Elasticsearch
func (m *MetricSet) extractAndTransform(ctx context.Context) ([]mb.Event, error) {
	extractedMetricsData, err := m.extract(ctx, m.extractor)
	if err != nil {
		return nil, fmt.Errorf("error extracting data %w", err)
	}
	return m.transform(extractedMetricsData), nil
}

// transform is the T of an ETL (refer to the 'extract' method above if you need to see the origin). Transforms the data
// to create a Kibana/Elasticsearch friendly JSON. Data from Oracle is pretty fragmented by design so a lot of data
// was necessary. Data is organized by sysmetric entity that contains metrics details.
func (m *MetricSet) transform(in *extractedData) []mb.Event {
	sysMetric := m.addSysmetricData(in.sysmetricMetrics)
	events := make([]mb.Event, 0)
	for _, v := range sysMetric {
		events = append(events, mb.Event{MetricSetFields: v})
	}
	return events
}
