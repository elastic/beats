// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package performance

import (
	"context"

	"github.com/elastic/beats/x-pack/metricbeat/module/oracle"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

// extract is the E of a ETL processing. Gets the data files, used/free space and temp free space data that is fetch
// by doing queries to Oracle
func (m *MetricSet) extract(ctx context.Context, extractor performanceExtractMethods) (out *extractedData, err error) {
	out = &extractedData{}

	if out.bufferCacheHitRatios, err = extractor.bufferCacheHitRatio(ctx); err != nil {
		return nil, errors.Wrap(err, "error getting buffer cache hit ratio")
	}

	if out.libraryData, err = extractor.library(ctx); err != nil {
		return nil, errors.Wrap(err, "error getting library data")
	}

	if out.cursorsByUsernameAndMachine, err = extractor.cursorsByUsernameAndMachine(ctx); err != nil {
		return nil, errors.Wrap(err, "error getting cursors by username and machine")
	}

	if out.totalCursors, err = extractor.totalCursors(ctx); err != nil {
		return nil, errors.Wrap(err, "error getting total cursors")
	}

	return
}

func (m *MetricSet) extractAndTransform(ctx context.Context) ([]mb.Event, error) {
	extractedMetricsData, err := m.extract(ctx, m.extractor)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting data")
	}

	return m.transform(extractedMetricsData), nil
}

// transform is the T of an ETL (refer to the 'extract' method above if you need to see the origin). Transforms the data
// to create a Kibana/Elasticsearch friendly JSON. Data from Oracle is pretty fragmented by design so a lot of data
// was necessary. Data is organized by Tablespace entity (Tablespaces might contain one or more data files)
func (m *MetricSet) transform(in *extractedData) []mb.Event {
	bufferCache := m.addBufferCacheRatioData(in.bufferCacheHitRatios)
	cursorByUsernameAndMachineEvents := m.addCursorByUsernameAndMachine(in.cursorsByUsernameAndMachine)

	cursorEvent := m.addCursorData(in.totalCursors)
	cursorEvent.Update(m.addLibraryData(in.libraryData))

	events := make([]mb.Event, 0)

	events = append(events, mb.Event{MetricSetFields: cursorEvent})

	for _, v := range cursorByUsernameAndMachineEvents {
		events = append(events, mb.Event{MetricSetFields: v})
	}

	for _, v := range bufferCache {
		events = append(events, mb.Event{MetricSetFields: v})
	}

	return events
}

func (m *MetricSet) addCursorData(cs *totalCursors) common.MapStr {
	out := make(common.MapStr)

	oracle.SetSqlValue(m.Logger(), out, "cursors.opened.total", &oracle.Int64Value{NullInt64: cs.totalCursors})
	oracle.SetSqlValue(m.Logger(), out, "cursors.opened.current", &oracle.Int64Value{NullInt64: cs.currentCursors})
	oracle.SetSqlValue(m.Logger(), out, "cursors.session.cache_hits", &oracle.Int64Value{NullInt64: cs.sessCurCacheHits})
	oracle.SetSqlValue(m.Logger(), out, "cursors.parse.total", &oracle.Int64Value{NullInt64: cs.parseCountTotal})
	oracle.SetSqlValue(m.Logger(), out, "cursors.total.cache_hit.pct", &oracle.Float64Value{NullFloat64: cs.cacheHitsTotalCursorsRatio})
	oracle.SetSqlValue(m.Logger(), out, "cursors.parse.real", &oracle.Int64Value{NullInt64: cs.realParses})

	return out
}

func (m *MetricSet) addCursorByUsernameAndMachine(cs []cursorsByUsernameAndMachine) []common.MapStr {
	out := make([]common.MapStr, 0)

	for _, v := range cs {
		ms := common.MapStr{}

		oracle.SetSqlValue(m.Logger(), ms, "username", &oracle.StringValue{NullString: v.username})
		oracle.SetSqlValue(m.Logger(), ms, "machine", &oracle.StringValue{NullString: v.machine})
		oracle.SetSqlValue(m.Logger(), ms, "cursors.total", &oracle.Int64Value{NullInt64: v.total})
		oracle.SetSqlValue(m.Logger(), ms, "cursors.max", &oracle.Int64Value{NullInt64: v.max})
		oracle.SetSqlValue(m.Logger(), ms, "cursors.avg", &oracle.Float64Value{NullFloat64: v.avg})

		out = append(out, ms)
	}

	return out
}

func (m *MetricSet) addLibraryData(ls []library) common.MapStr {
	out := common.MapStr{}

	for _, v := range ls {
		if v.name.Valid {
			oracle.SetSqlValue(m.Logger(), out, v.name.String, &oracle.Float64Value{NullFloat64: v.value})
		}
	}

	return out
}

// addTempFreeSpaceData is specific to the TEMP Tablespace.
func (m *MetricSet) addBufferCacheRatioData(bs []bufferCacheHitRatio) map[string]common.MapStr {
	out := make(map[string]common.MapStr)

	for _, bufferCacheHitRatio := range bs {
		if _, found := out[bufferCacheHitRatio.name.String]; !found {
			out[bufferCacheHitRatio.name.String] = common.MapStr{}
		}

		_, _ = out[bufferCacheHitRatio.name.String].Put("buffer_pool", bufferCacheHitRatio.name.String)

		oracle.SetSqlValueWithParentKey(m.Logger(), out, bufferCacheHitRatio.name.String, "cache.buffer.hit.pct", &oracle.Float64Value{NullFloat64: bufferCacheHitRatio.hitRatio})
		oracle.SetSqlValueWithParentKey(m.Logger(), out, bufferCacheHitRatio.name.String, "cache.get.consistent", &oracle.Int64Value{NullInt64: bufferCacheHitRatio.consistentGets})
		oracle.SetSqlValueWithParentKey(m.Logger(), out, bufferCacheHitRatio.name.String, "cache.get.db_blocks", &oracle.Int64Value{NullInt64: bufferCacheHitRatio.dbBlockGets})
		oracle.SetSqlValueWithParentKey(m.Logger(), out, bufferCacheHitRatio.name.String, "cache.physical_reads", &oracle.Int64Value{NullInt64: bufferCacheHitRatio.physicalReads})

	}

	return out
}
