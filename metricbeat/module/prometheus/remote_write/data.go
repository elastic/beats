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

package remote_write

import (
	"math"

	"github.com/prometheus/common/model"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// DefaultRemoteWriteEventsGeneratorFactory returns the default prometheus events generator
func DefaultRemoteWriteEventsGeneratorFactory(ms mb.BaseMetricSet, metricsCount bool) (RemoteWriteEventsGenerator, error) {
	return &remoteWriteEventGenerator{metricsCount: metricsCount}, nil
}

type remoteWriteEventGenerator struct {
	metricsCount bool
}

func (p *remoteWriteEventGenerator) Start() {}
func (p *remoteWriteEventGenerator) Stop()  {}

func (p *remoteWriteEventGenerator) GenerateEvents(metrics model.Samples) map[string]mb.Event {
	eventList := map[string]mb.Event{}

	for _, metric := range metrics {
		labels := mapstr.M{}

		if metric == nil {
			continue
		}
		val := float64(metric.Value)
		if math.IsNaN(val) || math.IsInf(val, 0) {
			continue
		}

		//nolint:typecheck,nolintlint // 'name' is being used in as a key in mapstr.M below
		name := string(metric.Metric["__name__"])
		delete(metric.Metric, "__name__")

		for k, v := range metric.Metric {
			labels[string(k)] = v
		}

		// join metrics with same labels and same timestamp in a single event
		labelsHash := labels.String() + metric.Timestamp.Time().String()
		if _, ok := eventList[labelsHash]; !ok {
			eventList[labelsHash] = mb.Event{
				RootFields: make(mapstr.M),
				ModuleFields: mapstr.M{
					"metrics": mapstr.M{},
				},
				Timestamp: metric.Timestamp.Time(),
			}

			// Add labels
			if len(labels) > 0 {
				eventList[labelsHash].ModuleFields["labels"] = labels
			}
		}

		// Not checking anything here because we create these maps some lines before
		e := eventList[labelsHash]

		data := mapstr.M{name: val}
		e.ModuleFields["metrics"].(mapstr.M).Update(data)
	}

	if p.metricsCount {
		for _, e := range eventList {
			// In x-pack prometheus module, the metrics are nested under the "prometheus" key directly.
			// whereas in non-x-pack prometheus module, the metrics are nested under the "prometheus.metrics" key.
			// Also, it is important that we do not just increment by 1 for each e.ModuleFields["metrics"] may have more than 1 metric.
			// See unit tests for the same.
			v, ok := e.ModuleFields["metrics"].(mapstr.M)
			if ok {
				e.RootFields["metrics_count"] = len(v)
			}
		}
	}

	return eventList
}
