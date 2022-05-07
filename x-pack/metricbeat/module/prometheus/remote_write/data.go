// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote_write

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/model"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/prometheus/remote_write"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/prometheus/collector"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	counterType   = "counter_type"
	histogramType = "histogram_type"
	otherType     = "other_type"
)

type histogram struct {
	timestamp  time.Time
	buckets    []*dto.Bucket
	labels     mapstr.M
	metricName string
}

func remoteWriteEventsGeneratorFactory(base mb.BaseMetricSet) (remote_write.RemoteWriteEventsGenerator, error) {
	var err error
	config := defaultConfig
	if err = base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if config.UseTypes {
		// use a counter cache with a timeout of 5x the period, as a safe value
		// to make sure that all counters are available between fetches
		counters := collector.NewCounterCache(base.Module().Config().Period * 5)

		g := remoteWriteTypedGenerator{
			counterCache: counters,
			rateCounters: config.RateCounters,
		}

		g.counterPatterns, err = p.CompilePatternList(config.TypesPatterns.CounterPatterns)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to compile counter patterns")
		}
		g.histogramPatterns, err = p.CompilePatternList(config.TypesPatterns.HistogramPatterns)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to compile histogram patterns")
		}

		return &g, nil
	}

	return remote_write.DefaultRemoteWriteEventsGeneratorFactory(base)
}

type remoteWriteTypedGenerator struct {
	counterCache      collector.CounterCache
	rateCounters      bool
	counterPatterns   []*regexp.Regexp
	histogramPatterns []*regexp.Regexp
}

func (g *remoteWriteTypedGenerator) Start() {
	cfgwarn.Beta("Prometheus 'use_types' setting is beta")

	if g.rateCounters {
		cfgwarn.Experimental("Prometheus 'rate_counters' setting is experimental")
	}

	g.counterCache.Start()
}

func (g *remoteWriteTypedGenerator) Stop() {
	logp.Debug("prometheus.remote_write.cache", "stopping counterCache")
	g.counterCache.Stop()
}

// GenerateEvents receives a list of Sample and:
// 1. guess the type of the sample metric
// 2. handle it properly using "types" logic
// 3. if metrics of histogram type then it is converted to ES histogram
// 4. metrics with the same set of labels are grouped into same events
func (g remoteWriteTypedGenerator) GenerateEvents(metrics model.Samples) map[string]mb.Event {
	var data mapstr.M
	histograms := map[string]histogram{}
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

		name := string(metric.Metric["__name__"])
		delete(metric.Metric, "__name__")

		for k, v := range metric.Metric {
			labels[string(k)] = v
		}

		promType := g.findMetricType(name, labels)

		labelsHash := labels.String() + metric.Timestamp.Time().String()
		labelsClone := labels.Clone()
		labelsClone.Delete("le")
		if promType == histogramType {
			labelsHash = labelsClone.String() + metric.Timestamp.Time().String()
		}
		// join metrics with same labels in a single event
		if _, ok := eventList[labelsHash]; !ok {
			eventList[labelsHash] = mb.Event{
				ModuleFields: mapstr.M{},
				Timestamp:    metric.Timestamp.Time(),
			}

			// Add labels
			if len(labels) > 0 {
				if promType == histogramType {
					eventList[labelsHash].ModuleFields["labels"] = labelsClone
				} else {
					eventList[labelsHash].ModuleFields["labels"] = labels
				}
			}
		}

		e := eventList[labelsHash]
		switch promType {
		case counterType:
			data = mapstr.M{
				name: g.rateCounterFloat64(name, labels, val),
			}
		case otherType:
			data = mapstr.M{
				name: mapstr.M{
					"value": val,
				},
			}
		case histogramType:
			histKey := name + labelsClone.String()

			le, _ := labels.GetValue("le")
			upperBound := string(le.(model.LabelValue))

			bucket, err := strconv.ParseFloat(upperBound, 64)
			if err != nil {
				continue
			}
			v := uint64(val)
			b := &dto.Bucket{
				CumulativeCount: &v,
				UpperBound:      &bucket,
			}
			hist, ok := histograms[histKey]
			if !ok {
				hist = histogram{}
			}
			hist.buckets = append(hist.buckets, b)
			hist.timestamp = metric.Timestamp.Time()
			hist.labels = labelsClone
			hist.metricName = name
			histograms[histKey] = hist
			continue
		}
		e.ModuleFields.Update(data)

	}

	// process histograms together
	g.processPromHistograms(eventList, histograms)
	return eventList
}

// rateCounterUint64 fills a counter value and optionally adds the rate if rate_counters is enabled
func (g *remoteWriteTypedGenerator) rateCounterUint64(name string, labels mapstr.M, value uint64) mapstr.M {
	d := mapstr.M{
		"counter": value,
	}

	if g.rateCounters {
		d["rate"], _ = g.counterCache.RateUint64(name+labels.String(), value)
	}

	return d
}

// rateCounterFloat64 fills a counter value and optionally adds the rate if rate_counters is enabled
func (g *remoteWriteTypedGenerator) rateCounterFloat64(name string, labels mapstr.M, value float64) mapstr.M {
	d := mapstr.M{
		"counter": value,
	}
	if g.rateCounters {
		d["rate"], _ = g.counterCache.RateFloat64(name+labels.String(), value)
	}

	return d
}

// processPromHistograms receives a group of Histograms and converts each one to ES histogram
func (g *remoteWriteTypedGenerator) processPromHistograms(eventList map[string]mb.Event, histograms map[string]histogram) {
	for _, histogram := range histograms {
		labelsHash := histogram.labels.String() + histogram.timestamp.String()
		if _, ok := eventList[labelsHash]; !ok {
			eventList[labelsHash] = mb.Event{
				ModuleFields: mapstr.M{},
				Timestamp:    histogram.timestamp,
			}

			// Add labels
			if len(histogram.labels) > 0 {
				eventList[labelsHash].ModuleFields["labels"] = histogram.labels
			}
		}

		e := eventList[labelsHash]

		hist := dto.Histogram{
			Bucket: histogram.buckets,
		}
		name := strings.TrimSuffix(histogram.metricName, "_bucket")
		data := mapstr.M{
			name: mapstr.M{
				"histogram": collector.PromHistogramToES(g.counterCache, histogram.metricName, histogram.labels, &hist),
			},
		}
		e.ModuleFields.Update(data)
	}
}

// findMetricType evaluates the type of the metric by check the metricname format in order to handle it properly
func (g *remoteWriteTypedGenerator) findMetricType(metricName string, labels mapstr.M) string {
	leLabel := false
	if _, ok := labels["le"]; ok {
		leLabel = true
	}

	// handle user provided patterns
	if len(g.counterPatterns) > 0 {
		if p.MatchMetricFamily(metricName, g.counterPatterns) {
			return counterType
		}
	}
	if len(g.histogramPatterns) > 0 {
		if p.MatchMetricFamily(metricName, g.histogramPatterns) && leLabel {
			return histogramType
		}
	}

	// handle defaults
	if strings.HasSuffix(metricName, "_total") || strings.HasSuffix(metricName, "_sum") ||
		strings.HasSuffix(metricName, "_count") {
		return counterType
	} else if strings.HasSuffix(metricName, "_bucket") && leLabel {
		return histogramType
	}

	return otherType
}
