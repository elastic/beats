// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
