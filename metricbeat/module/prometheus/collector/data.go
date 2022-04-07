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

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/metricbeat/helper/labelhash"
	"github.com/elastic/beats/v8/metricbeat/mb"

	dto "github.com/prometheus/client_model/go"
)

// PromEvent stores a set of one or more metrics with the same labels
type PromEvent struct {
	Data   common.MapStr
	Labels common.MapStr
}

// LabelsHash returns a repeatable string that is unique for the set of labels in this event
func (p *PromEvent) LabelsHash() string {
	return labelhash.LabelHash(p.Labels)
}

// DefaultPromEventsGeneratorFactory returns the default prometheus events generator
func DefaultPromEventsGeneratorFactory(ms mb.BaseMetricSet) (PromEventsGenerator, error) {
	return &promEventGenerator{}, nil
}

type promEventGenerator struct{}

func (p *promEventGenerator) Start() {}
func (p *promEventGenerator) Stop()  {}

// GeneratePromEvents DefaultPromEventsGenerator stores all Prometheus metrics using
// only double field type in Elasticsearch.
func (p *promEventGenerator) GeneratePromEvents(mf *dto.MetricFamily) []PromEvent {
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
			if !math.IsNaN(counter.GetValue()) && !math.IsInf(counter.GetValue(), 0) {
				events = append(events, PromEvent{
					Data: common.MapStr{
						"metrics": common.MapStr{
							name: counter.GetValue(),
						},
					},
					Labels: labels,
				})
			}
		}

		gauge := metric.GetGauge()
		if gauge != nil {
			if !math.IsNaN(gauge.GetValue()) && !math.IsInf(gauge.GetValue(), 0) {
				events = append(events, PromEvent{
					Data: common.MapStr{
						"metrics": common.MapStr{
							name: gauge.GetValue(),
						},
					},
					Labels: labels,
				})
			}
		}

		summary := metric.GetSummary()
		if summary != nil {
			if !math.IsNaN(summary.GetSampleSum()) && !math.IsInf(summary.GetSampleSum(), 0) {
				events = append(events, PromEvent{
					Data: common.MapStr{
						"metrics": common.MapStr{
							name + "_sum":   summary.GetSampleSum(),
							name + "_count": summary.GetSampleCount(),
						},
					},
					Labels: labels,
				})
			}

			for _, quantile := range summary.GetQuantile() {
				if math.IsNaN(quantile.GetValue()) || math.IsInf(quantile.GetValue(), 0) {
					continue
				}

				quantileLabels := labels.Clone()
				quantileLabels["quantile"] = strconv.FormatFloat(quantile.GetQuantile(), 'f', -1, 64)
				events = append(events, PromEvent{
					Data: common.MapStr{
						"metrics": common.MapStr{
							name: quantile.GetValue(),
						},
					},
					Labels: quantileLabels,
				})
			}
		}

		histogram := metric.GetHistogram()
		if histogram != nil {
			if !math.IsNaN(histogram.GetSampleSum()) && !math.IsInf(histogram.GetSampleSum(), 0) {
				events = append(events, PromEvent{
					Data: common.MapStr{
						"metrics": common.MapStr{
							name + "_sum":   histogram.GetSampleSum(),
							name + "_count": histogram.GetSampleCount(),
						},
					},
					Labels: labels,
				})
			}

			for _, bucket := range histogram.GetBucket() {
				if bucket.GetCumulativeCount() == uint64(math.NaN()) || bucket.GetCumulativeCount() == uint64(math.Inf(0)) {
					continue
				}

				bucketLabels := labels.Clone()
				bucketLabels["le"] = strconv.FormatFloat(bucket.GetUpperBound(), 'f', -1, 64)

				events = append(events, PromEvent{
					Data: common.MapStr{
						"metrics": common.MapStr{
							name + "_bucket": bucket.GetCumulativeCount(),
						},
					},
					Labels: bucketLabels,
				})
			}
		}

		untyped := metric.GetUntyped()
		if untyped != nil {
			if !math.IsNaN(untyped.GetValue()) && !math.IsInf(untyped.GetValue(), 0) {
				events = append(events, PromEvent{
					Data: common.MapStr{
						"metrics": common.MapStr{
							name: untyped.GetValue(),
						},
					},
					Labels: labels,
				})
			}
		}
	}
	return events
}
