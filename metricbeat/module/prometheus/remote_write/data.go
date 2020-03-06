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

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

func samplesToEvents(metrics model.Samples) map[string]mb.Event {
	eventList := map[string]mb.Event{}

	for _, metric := range metrics {
		labels := common.MapStr{}

		if metric == nil {
			continue
		}
		name := string(metric.Metric["__name__"])
		delete(metric.Metric, "__name__")

		metric.Metric.Fingerprint()
		for k, v := range metric.Metric {
			labels[string(k)] = v
		}

		val := float64(metric.Value)
		if !math.IsNaN(val) && !math.IsInf(val, 0) {
			// join metrics with same labels in a single event
			labelsHash := labels.String()
			if _, ok := eventList[labelsHash]; !ok {
				eventList[labelsHash] = mb.Event{
					ModuleFields: common.MapStr{
						"metrics": common.MapStr{},
					},
				}

				// Add labels
				if len(labels) > 0 {
					eventList[labelsHash].ModuleFields["labels"] = labels
				}
			}

			// Not checking anything here because we create these maps some lines before
			e := eventList[labelsHash]
			e.Timestamp = metric.Timestamp.Time()
			data := common.MapStr{
				name: val,
			}
			e.ModuleFields["metrics"].(common.MapStr).Update(data)
		}
	}

	return eventList
}
