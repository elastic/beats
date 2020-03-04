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

// TODO this file is entirely copied from collector/data.go, refactor this

import (
	"math"
	"time"

	"github.com/prometheus/common/model"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// PromEvent stores a set of one or more metrics with the same labels
type PromEvent struct {
	data      common.MapStr
	labels    common.MapStr
	timestamp time.Time
}

// LabelsHash returns a repeatable string that is unique for the set of labels in this event
func (p *PromEvent) LabelsHash() string {
	return p.labels.String()
}

func samplesToEvents(metrics model.Samples) map[string]mb.Event {
	var events []PromEvent

	for _, metric := range metrics {
		labels := common.MapStr{}

		if metric == nil {
			continue
		}
		// TODO this may be nil?
		name := string(metric.Metric["__name__"])
		delete(metric.Metric, "__name__")

		for k, v := range metric.Metric {
			labels[string(k)] = v
		}

		val := float64(metric.Value)
		if !math.IsNaN(val) && !math.IsInf(val, 0) {
			events = append(events, PromEvent{
				data: common.MapStr{
					name: val,
				},
				labels:    labels,
				timestamp: metric.Timestamp.Time(),
			})
		}
	}

	// join metrics with same labels in a single event
	eventList := map[string]mb.Event{}

	for _, promEvent := range events {
		labelsHash := promEvent.LabelsHash()
		if _, ok := eventList[labelsHash]; !ok {
			eventList[labelsHash] = mb.Event{
				ModuleFields: common.MapStr{
					"metrics": common.MapStr{},
				},
			}

			// Add labels
			if len(promEvent.labels) > 0 {
				eventList[labelsHash].ModuleFields["labels"] = promEvent.labels
			}
		}

		// Not checking anything here because we create these maps some lines before
		e := eventList[labelsHash]
		e.Timestamp = promEvent.timestamp
		e.ModuleFields["metrics"].(common.MapStr).Update(promEvent.data)
	}

	return eventList
}
