// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package collector

import (
	"math"
	"strconv"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/prometheus/collector"
	"github.com/elastic/elastic-agent-libs/mapstr"

	dto "github.com/prometheus/client_model/go"
)

func promEventsGeneratorFactory(base mb.BaseMetricSet) (collector.PromEventsGenerator, error) {
	config := config{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if config.UseTypes {
		// use a counter cache with a timeout of 5x the period, as a safe value
		// to make sure that all counters are available between fetches
		counters := NewCounterCache(base.Module().Config().Period * 5)

		g := typedGenerator{
			counterCache: counters,
			rateCounters: config.RateCounters,
		}

		return &g, nil
	}

	return collector.DefaultPromEventsGeneratorFactory(base)
}

type typedGenerator struct {
	counterCache CounterCache
	rateCounters bool
}

func (g *typedGenerator) Start() {
	cfgwarn.Beta("Prometheus 'use_types' setting is beta")

	if g.rateCounters {
		cfgwarn.Experimental("Prometheus 'rate_counters' setting is experimental")
	}

	g.counterCache.Start()
}

func (g *typedGenerator) Stop() {
	logp.Debug("prometheus.collector.cache", "stopping counterCache")
	g.counterCache.Stop()
}

// GeneratePromEvents stores all Prometheus metrics using
// specific Elasticsearch data types.
func (g *typedGenerator) GeneratePromEvents(mf *dto.MetricFamily) []collector.PromEvent {
	var events []collector.PromEvent

	name := *mf.Name
	metrics := mf.Metric
	for _, metric := range metrics {
		labels := mapstr.M{}

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
					Data: mapstr.M{
						name: g.rateCounterFloat64(name, labels, counter.GetValue()),
					},
					Labels: labels,
				})
			}
		}

		gauge := metric.GetGauge()
		if gauge != nil {
			if !math.IsNaN(gauge.GetValue()) && !math.IsInf(gauge.GetValue(), 0) {
				events = append(events, collector.PromEvent{
					Data: mapstr.M{
						name: mapstr.M{
							"value": gauge.GetValue(),
						},
					},
					Labels: labels,
				})
			}
		}

		summary := metric.GetSummary()
		if summary != nil {
			if !math.IsNaN(summary.GetSampleSum()) && !math.IsInf(summary.GetSampleSum(), 0) {
				events = append(events, collector.PromEvent{
					Data: mapstr.M{
						name + "_sum":   g.rateCounterFloat64(name, labels, summary.GetSampleSum()),
						name + "_count": g.rateCounterUint64(name, labels, summary.GetSampleCount()),
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
					Data: mapstr.M{
						name: mapstr.M{
							"value": quantile.GetValue(),
						},
					},
					Labels: quantileLabels,
				})
			}
		}

		histogram := metric.GetHistogram()
		if histogram != nil {
			events = append(events, collector.PromEvent{
				Data: mapstr.M{
					name: mapstr.M{
						"histogram": PromHistogramToES(g.counterCache, name, labels, histogram),
					},
				},
				Labels: labels,
			})
			/*
				TODO convert histogram to ES type
				Send sum & count? not sure it's worth it
			*/
		}

		untyped := metric.GetUntyped()
		if untyped != nil {
			if !math.IsNaN(untyped.GetValue()) && !math.IsInf(untyped.GetValue(), 0) {
				events = append(events, collector.PromEvent{
					Data: mapstr.M{
						name: mapstr.M{
							"value": untyped.GetValue(),
						},
					},
					Labels: labels,
				})
			}
		}
	}
	return events
}

// rateCounterUint64 fills a counter value and optionally adds the rate if rate_counters is enabled
func (g *typedGenerator) rateCounterUint64(name string, labels mapstr.M, value uint64) mapstr.M {
	d := mapstr.M{
		"counter": value,
	}

	if g.rateCounters {
		d["rate"], _ = g.counterCache.RateUint64(name+labels.String(), value)
	}

	return d
}

// rateCounterFloat64 fills a counter value and optionally adds the rate if rate_counters is enabled
func (g *typedGenerator) rateCounterFloat64(name string, labels mapstr.M, value float64) mapstr.M {
	d := mapstr.M{
		"counter": value,
	}

	if g.rateCounters {
		d["rate"], _ = g.counterCache.RateFloat64(name+labels.String(), value)
	}

	return d
}
