// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package performance

import (
	"context"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/metricbeat/mb"
)

// extract is the E of a ETL processing. Gets all the performance information into an instance of extractedData
func (m *MetricSet) extract(ctx context.Context, extractor performanceExtractMethods) (out *extractedData, err error) {
	out = &extractedData{}

	if out.bufferCacheHitRatios, err = extractor.bufferCacheHitRatio(ctx); err != nil {
		return nil, errors.Wrap(err, "error getting buffer cache hit ratio")
	}

	if out.libraryData, err = extractor.libraryCache(ctx); err != nil {
		return nil, errors.Wrap(err, "error getting libraryCache data")
	}

	if out.cursorsByUsernameAndMachine, err = extractor.cursorsByUsernameAndMachine(ctx); err != nil {
		return nil, errors.Wrap(err, "error getting cursors by username and machine")
	}

	if out.totalCursors, err = extractor.totalCursors(ctx); err != nil {
		return nil, errors.Wrap(err, "error getting total cursors")
	}

	return
}

// extractAndTransform just composes the ET operations from a ETL. It's called by the Fetch method, which is the one
// that "loads" the data into Elasticsearch
func (m *MetricSet) extractAndTransform(ctx context.Context) ([]mb.Event, error) {
	extractedMetricsData, err := m.extract(ctx, m.extractor)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting data")
	}

	return m.transform(extractedMetricsData), nil
}

// transform is the T of an ETL (refer to the 'extract' method above if you need to see the origin). Transforms the data
// to create a Kibana/Elasticsearch friendly JSON. Data from Oracle is pretty fragmented by design so a lot of data
// was necessary. More than one different event is generated. Refer to the _meta folder too see ones.
func (m *MetricSet) transform(in *extractedData) []mb.Event {
	bufferCache := m.addBufferCacheRatioData(in.bufferCacheHitRatios)
	cursorByUsernameAndMachineEvents := m.addCursorByUsernameAndMachine(in.cursorsByUsernameAndMachine)

	cursorEvent := m.addCursorData(in.totalCursors)
	cursorEvent.Update(m.addLibraryCacheData(in.libraryData))

	events := make([]mb.Event, 0)

	for _, v := range bufferCache {
		events = append(events, mb.Event{MetricSetFields: v})
	}

	events = append(events, mb.Event{MetricSetFields: cursorEvent})

	for _, v := range cursorByUsernameAndMachineEvents {
		events = append(events, mb.Event{MetricSetFields: v})
	}

	return events
}
