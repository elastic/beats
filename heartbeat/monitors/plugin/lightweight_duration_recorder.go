// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package plugin

import (
	"strconv"

	"github.com/elastic/beats/v7/libbeat/monitoring"
)

type DurationRegistryRecorder interface {
	RecordDuration(duration int64)
}

type durationRecorder struct {
	durationHistogram *monitoring.UniqueList // histogram with the count of durations (in ms) for tests
}

func NewDurationRecorder(pluginName string, r *monitoring.Registry) DurationRegistryRecorder {
	pluginRegistry := r.GetRegistry(pluginName)
	if pluginRegistry == nil {
		pluginRegistry = r.NewRegistry(pluginName)
	}

	durationHistogram := monitoring.NewUniqueList()
	monitoring.NewFunc(pluginRegistry, "duration_histogram", durationHistogram.ReportMap, monitoring.Report)

	return durationRecorder{durationHistogram}
}

func (dr durationRecorder) RecordDuration(d int64) {
	dr.durationHistogram.Add(strconv.FormatInt(d, 10))
}
