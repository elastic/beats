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

package input_logfile

import "github.com/elastic/beats/v7/libbeat/monitoring"

// readerMetrics contains the basic summary of file readers in filestream input.
type readerMetrics struct {
	reg     *monitoring.Registry
	started *monitoring.Int
	stopped *monitoring.Int
	running *monitoring.Int
	failed  *monitoring.Int
}

func newReaderMetrics(pluginName string) *readerMetrics {
	r := monitoring.Default.NewRegistry("filebeat." + pluginName + ".readers")
	return &readerMetrics{
		reg:     r,
		started: monitoring.NewInt(r, "started"),
		stopped: monitoring.NewInt(r, "stopped"),
		running: monitoring.NewInt(r, "running"),
		failed:  monitoring.NewInt(r, "failed"),
	}
}

func (r *readerMetrics) onReaderStarted() {
	r.started.Inc()
	r.running.Inc()
}

func (r *readerMetrics) onReaderStopped() {
	r.stopped.Inc()
	r.running.Dec()
}

func (r *readerMetrics) onReaderGroupStopped(count int64) {
	r.stopped.Add(count)
	r.running.Sub(count)
}

func (r *readerMetrics) onReaderFailed() {
	r.failed.Inc()
}
