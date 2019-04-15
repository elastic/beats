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

package collector

import (
	"math"
	"strconv"

	"github.com/elastic/beats/libbeat/common"

	dto "github.com/prometheus/client_model/go"
)

// PromEvent stores a set of one or more metrics with the same labels
type PromEvent struct {
	data   common.MapStr
	labels common.MapStr
}

// LabelsHash returns a repeatable string that is unique for the set of labels in this event
func (p *PromEvent) LabelsHash() string {
	return p.labels.String()
}

func getPromEventsFromMetricFamily(mf *dto.MetricFamily) []PromEvent {
	var events []PromEvent

	name := *mf.Name
	metrics := mf.Metric
	for _, metric := range metrics {
		labels := common.MapStr{}

		if len(metric.Label) != 0 {
			for _, label := range metric.Label {
				if label.GetName() != "" && label.GetValue() != "" {
					labels[label.GetName()] = label.GetValue()
				}
			}
		}

		counter := metric.GetCounter()
		if counter != nil {
			events = append(events, PromEvent{
				data: common.MapStr{
					name: counter.GetValue(),
				},
				labels: labels,
			})
		}

		gauge := metric.GetGauge()
		if gauge != nil {
			events = append(events, PromEvent{
				data: common.MapStr{
					name: gauge.GetValue(),
				},
				labels: labels,
			})
		}

		summary := metric.GetSummary()
		if summary != nil {
			events = append(events, PromEvent{
				data: common.MapStr{
					name + "_sum":   summary.GetSampleSum(),
					name + "_count": summary.GetSampleCount(),
				},
				labels: labels,
			})

			for _, quantile := range summary.GetQuantile() {
				if math.IsNaN(quantile.GetValue()) {
					continue
				}

				quantileLabels := labels.Clone()
				quantileLabels["quantile"] = strconv.FormatFloat(quantile.GetQuantile(), 'f', -1, 64)
				events = append(events, PromEvent{
					data: common.MapStr{
						name: quantile.GetValue(),
					},
					labels: quantileLabels,
				})
			}
		}

		histogram := metric.GetHistogram()
		if histogram != nil {
			events = append(events, PromEvent{
				data: common.MapStr{
					name + "_sum":   histogram.GetSampleSum(),
					name + "_count": histogram.GetSampleCount(),
				},
				labels: labels,
			})

			for _, bucket := range histogram.GetBucket() {
				bucketLabels := labels.Clone()
				bucketLabels["le"] = strconv.FormatFloat(bucket.GetUpperBound(), 'f', -1, 64)

				events = append(events, PromEvent{
					data: common.MapStr{
						name + "_bucket": bucket.GetCumulativeCount(),
					},
					labels: bucketLabels,
				})
			}
		}

		untyped := metric.GetUntyped()
		if untyped != nil {
			events = append(events, PromEvent{
				data: common.MapStr{
					name: untyped.GetValue(),
				},
				labels: labels,
			})
		}
	}
	return events
}
