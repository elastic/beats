// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package collector

import (
	"math"
	"strconv"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/prometheus/collector"

	dto "github.com/prometheus/client_model/go"
)

func promEventsGeneratorFactory(base mb.BaseMetricSet) (collector.PromEventsGenerator, error) {
	config := config{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if config.UseTypes {
		if config.RateCounters {

		}

		return promEventsGenerator, nil
	}

	return collector.DefaultPromEventsGenerator, nil
}

// promEventsGenerator stores all Prometheus metrics using
// only double field type in Elasticsearch.
func promEventsGenerator(mf *dto.MetricFamily) []collector.PromEvent {
	var events []collector.PromEvent

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
				events = append(events, collector.PromEvent{
					Data: common.MapStr{
						name + ".counter": counter.GetValue(),
					},
					Labels: labels,
				})
			}
		}

		gauge := metric.GetGauge()
		if gauge != nil {
			if !math.IsNaN(gauge.GetValue()) && !math.IsInf(gauge.GetValue(), 0) {
				events = append(events, collector.PromEvent{
					Data: common.MapStr{
						name: gauge.GetValue(),
					},
					Labels: labels,
				})
			}
		}

		summary := metric.GetSummary()
		if summary != nil {
			if !math.IsNaN(summary.GetSampleSum()) && !math.IsInf(summary.GetSampleSum(), 0) {
				events = append(events, collector.PromEvent{
					Data: common.MapStr{
						name + "_sum":   summary.GetSampleSum(),
						name + "_count": summary.GetSampleCount(),
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
				events = append(events, collector.PromEvent{
					Data: common.MapStr{
						name: quantile.GetValue(),
					},
					Labels: quantileLabels,
				})
			}
		}

		histogram := metric.GetHistogram()
		if histogram != nil {
			if !math.IsNaN(histogram.GetSampleSum()) && !math.IsInf(histogram.GetSampleSum(), 0) {
				events = append(events, collector.PromEvent{
					Data: common.MapStr{
						name + "_sum":   histogram.GetSampleSum(),
						name + "_count": histogram.GetSampleCount(),
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

				events = append(events, collector.PromEvent{
					Data: common.MapStr{
						name + "_bucket": bucket.GetCumulativeCount(),
					},
					Labels: bucketLabels,
				})
			}
		}

		untyped := metric.GetUntyped()
		if untyped != nil {
			if !math.IsNaN(untyped.GetValue()) && !math.IsInf(untyped.GetValue(), 0) {
				events = append(events, collector.PromEvent{
					Data: common.MapStr{
						name: untyped.GetValue(),
					},
					Labels: labels,
				})
			}
		}
	}
	return events
}
