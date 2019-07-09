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
		return nil, errors.Wrap(err, "error getting data_files")
	}

	if out.libraryData, err = extractor.library(ctx); err != nil {
		return nil, errors.Wrap(err, "error getting temp_free_space")
	}

	return
}

// transform is the T of an ETL (refer to the 'extract' method above if you need to see the origin). Transforms the data
// to create a Kibana/Elasticsearch friendly JSON. Data from Oracle is pretty fragmented by design so a lot of data
// was necessary. Data is organized by Tablespace entity (Tablespaces might contain one or more data files)
func (m *MetricSet) transform(in *extractedData) (out map[string]common.MapStr) {
	out = make(map[string]common.MapStr, 0)

	m.addBufferCacheRatioData(in.bufferCacheHitRatios, out)

	return
}

func (m *MetricSet) extractAndTransform(ctx context.Context) ([]mb.Event, error) {
	extractedMetricsData, err := m.extract(ctx, m.extractor)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting data")
	}

	out := m.transform(extractedMetricsData)

	events := make([]mb.Event, 0)
	for _, v := range out {
		events = append(events, mb.Event{MetricSetFields: v})
	}

	return events, nil
}

// addTempFreeSpaceData is specific to the TEMP Tablespace.
func (m *MetricSet) addBufferCacheRatioData(bs []bufferCacheHitRatio, out map[string]common.MapStr) {
	for key, cm := range out {
		val, err := cm.GetValue("name")
		if err != nil {
			m.Logger().Debug("error getting tablespace name")
			continue
		}

		name := val.(string)
		if name == "TEMP" {
			for _, bufferCacheHitRatio := range bs {
				oracle.CheckNullSqlValue(m.Logger(), out, key, "cache.buffer.hit.pct", &oracle.Float64Value{bufferCacheHitRatio.hitRatio})
				oracle.CheckNullSqlValue(m.Logger(), out, key, "space.used.bytes", &oracle.Int64Value{bufferCacheHitRatio.consistentGets})
				oracle.CheckNullSqlValue(m.Logger(), out, key, "space.free.bytes", &oracle.Int64Value{bufferCacheHitRatio.dbBlockGets})
				oracle.CheckNullSqlValue(m.Logger(), out, key, "space.free.bytes", &oracle.Int64Value{bufferCacheHitRatio.physicalReads})
			}
		}
	}
}
