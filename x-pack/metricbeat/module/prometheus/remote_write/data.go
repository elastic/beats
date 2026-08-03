// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote_write

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/common/model"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	rw "github.com/elastic/beats/v7/metricbeat/module/prometheus/remote_write"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/prometheus/collector"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	counterType   = "counter_type"
	histogramType = "histogram_type"
	otherType     = "other_type"
)

type histogram struct {
	timestamp  time.Time
	buckets    []*p.Bucket
	labels     mapstr.M
	metricName string
}

func remoteWriteEventsGeneratorFactory(base mb.BaseMetricSet, opts ...rw.RemoteWriteEventsGeneratorOption) (rw.RemoteWriteEventsGenerator, error) {
	config, err := loadRemoteWriteModuleConfig(base.Module())
	if err != nil {
		return nil, err
	}

	if config.UseTypes {
		base.Logger().Named("prometheus.remote_write.cache").Debugf("Period for counter cache for remote_write: %v", config.Period.String())
		// use a counter cache with a timeout of 5x the period, as a safe value
		// to make sure that all counters are available between fetches
		counters := collector.NewCounterCache(config.Period * 5)

		g := &remoteWriteTypedGenerator{
			counterCache: counters,
			rateCounters: config.RateCounters,
			metricsCount: config.MetricsCount,
			logger:       base.Logger(),
			now:          time.Now,
		}
		if g.logger == nil {
			g.logger = logp.NewNopLogger()
		}
		if config.UseHistogramAssembler {
			g.assemblyConfig = config.HistogramAssembly.assemblyConfig()
			g.retainedFlushes = make(map[string]mb.Event)
			g.histogramMon = registerHistogramAssemblerMonitoring(base.Metrics())
			g.assembler = newHistogramAssembler(g.assemblyConfig, g.histogramMon)
		}

		var err error
		g.counterPatterns, err = p.CompilePatternList(config.TypesPatterns.CounterPatterns)
		if err != nil {
			return nil, fmt.Errorf("unable to compile counter patterns: %w", err)
		}
		g.histogramPatterns, err = p.CompilePatternList(config.TypesPatterns.HistogramPatterns)
		if err != nil {
			return nil, fmt.Errorf("unable to compile histogram patterns: %w", err)
		}

		return g, nil
	}

	return rw.DefaultRemoteWriteEventsGeneratorFactory(base, opts...)
}

type remoteWriteTypedGenerator struct {
	metricsCount      bool
	counterCache      collector.CounterCache
	rateCounters      bool
	counterPatterns   []*regexp.Regexp
	histogramPatterns []*regexp.Regexp
	logger            *logp.Logger

	assemblyConfig   histogramAssemblyConfig
	assembler        *histogramAssembler
	histogramMon     *histogramAssemblerMonitoring
	now              func() time.Time
	retainedFlushes  map[string]mb.Event
	retainedCapacity flushEventCapacity
}

func (g *remoteWriteTypedGenerator) RequiresOwnerLoop() bool {
	return g.assembler != nil
}

// HistogramAssemblyStats exposes assembler counters for tests and introspection.
// Operational counters are published under histogram_assembler.* on the metricset monitoring registry.
func (g *remoteWriteTypedGenerator) HistogramAssemblyStats() histogramAssemblerStats {
	if g.assembler == nil {
		return histogramAssemblerStats{}
	}
	s := g.assembler.statsSnapshot()
	s.RetainedHistograms = g.retainedCapacity.histograms
	s.RetainedBuckets = g.retainedCapacity.buckets
	return s
}

func (g *remoteWriteTypedGenerator) Start() {
	g.logger.Warn(cfgwarn.Beta("Prometheus 'use_types' setting is beta"))

	if g.rateCounters {
		g.logger.Warn(cfgwarn.Experimental("Prometheus 'rate_counters' setting is experimental"))
	}

	g.counterCache.Start()
}

func (g *remoteWriteTypedGenerator) Stop() {
	// Shutdown is account-and-drop only: reporter.Done does not guarantee publication,
	// so pending assembler state and unpublished flush retries are discarded here.
	var shutdownDropped uint64
	if g.assembler != nil {
		shutdownDropped = uint64(len(g.assembler.pending) + len(g.retainedFlushes))
		g.assembler.shutdown(g.now())
	}
	if g.assembler != nil && len(g.retainedFlushes) > 0 {
		g.assembler.dropRetainedEvents(len(g.retainedFlushes))
		g.retainedFlushes = make(map[string]mb.Event)
		g.retainedCapacity = flushEventCapacity{}
	}
	if g.histogramMon != nil {
		g.histogramMon.observeShutdownDropped(shutdownDropped)
	}
	if g.logger != nil {
		g.logger.Debug("stopping counterCache")
	}
	g.counterCache.Stop()
}

func (g *remoteWriteTypedGenerator) NextFlushInterval() time.Duration {
	if g.assembler == nil {
		return 0
	}
	min := g.assemblyConfig.QuietPeriod
	if g.assemblyConfig.HardTimeout < min {
		min = g.assemblyConfig.HardTimeout
	}
	interval := min / 2
	if interval < time.Second {
		interval = time.Second
	}
	return interval
}

func (g *remoteWriteTypedGenerator) RetainUnpublishedFlushEvents(events map[string]mb.Event) {
	if g.assembler == nil {
		return
	}
	if len(events) == 0 {
		g.retainedFlushes = make(map[string]mb.Event)
		g.retainedCapacity = flushEventCapacity{}
		return
	}
	maxH := g.assemblyConfig.MaxPendingHistograms
	maxB := g.assemblyConfig.MaxPendingBuckets

	retained := make(map[string]mb.Event, len(events))
	var capSum flushEventCapacity
	for _, key := range sortedEventKeys(events) {
		ev := events[key]
		evCap := capacityFromFlushEvent(ev)
		nextH := capSum.histograms + evCap.histograms
		nextB := capSum.buckets + evCap.buckets
		if nextH > maxH || nextB > maxB {
			if g.assembler != nil {
				g.assembler.stats.RetentionDrops += uint64(evCap.histograms)
				g.assembler.stats.RetentionBucketDrops += uint64(evCap.buckets)
			}
			continue
		}
		retained[key] = ev
		capSum.histograms = nextH
		capSum.buckets = nextB
	}
	g.retainedFlushes = retained
	g.retainedCapacity = capSum
}

func (g *remoteWriteTypedGenerator) CheckCapacity(metrics model.Samples) error {
	if g.assembler == nil {
		return nil
	}
	batch := g.classifyBucketSamples(metrics)
	impact := g.assembler.capacityImpact(g.now(), batch)
	if g.assembler.wouldExceedCapacity(impact, g.retainedCapacity) {
		if g.histogramMon != nil {
			g.histogramMon.observeCapacityRejection()
		}
		return rw.ErrRemoteWriteCapacityExceeded
	}
	return nil
}

func (g *remoteWriteTypedGenerator) FlushExpired(now time.Time) map[string]mb.Event {
	if g.assembler == nil {
		return nil
	}
	events := g.assembler.flushExpired(now, g.counterCache, g.metricsCount)
	for k, v := range g.retainedFlushes {
		events[k] = v
	}
	// Retained capacity leaves accounting while the owner loop publishes synchronously;
	// it is restored by RetainUnpublishedFlushEvents if publication fails.
	g.retainedFlushes = make(map[string]mb.Event)
	g.retainedCapacity = flushEventCapacity{}
	return events
}

// GenerateEvents receives a list of Sample and:
// 1. guess the type of the sample metric
// 2. handle it properly using "types" logic
// 3. histogram _bucket samples are buffered for cross-request assembly
// 4. metrics with the same set of labels are grouped into same events
//
// Non-histogram samples (_sum, _count, counters, gauges) are emitted immediately in this call.
// Delayed histogram events are emitted later via FlushExpired and therefore carry a different
// metric-name set (and metrics_names_fingerprint) than the immediate _sum/_count event for the
// same labels. metrics_count, when enabled, is computed per emitted mb.Event, not as a unified
// logical scrape count across immediate and flush paths.
func (g *remoteWriteTypedGenerator) GenerateEvents(metrics model.Samples) map[string]mb.Event {
	var data mapstr.M
	histograms := map[string]histogram{}
	eventList := map[string]mb.Event{}

	for _, metric := range metrics {
		if metric == nil {
			continue
		}

		name, labels, val, ts, ok := g.readSample(metric)
		if !ok {
			continue
		}

		promType := g.findMetricType(name, labels)

		labelsHash := labels.String() + ts.String()
		labelsClone := labels.Clone()
		_ = labelsClone.Delete("le")
		if promType == histogramType {
			labelsHash = labelsClone.String() + ts.String()
		}

		if promType == histogramType {
			le, _ := labels.GetValue("le")
			upperBound, ok := parseLEUpperBound(string(le.(model.LabelValue)))
			if !ok {
				continue
			}
			if g.assembler != nil {
				g.assembler.ingest(g.now(), bucketSample{
					bucketMetricName: name,
					labels:           labelsClone,
					timestamp:        ts,
					buckets: []bucketUpdate{{
						upperBound:      upperBound,
						cumulativeCount: val,
					}},
				})
			} else {
				histKey := name + labelsClone.String()
				hist := histograms[histKey]
				hist.buckets = append(hist.buckets, &p.Bucket{
					CumulativeCount: &val,
					UpperBound:      &upperBound,
				})
				hist.timestamp = ts
				hist.labels = labelsClone
				hist.metricName = name
				histograms[histKey] = hist
			}
			continue
		}

		// join metrics with same labels in a single event
		if _, ok := eventList[labelsHash]; !ok {
			eventList[labelsHash] = mb.Event{
				RootFields:   mapstr.M{},
				ModuleFields: mapstr.M{},
				Timestamp:    ts,
			}

			if len(labels) > 0 {
				eventList[labelsHash].ModuleFields["labels"] = labels
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
		}

		e.ModuleFields.Update(data)
	}

	if g.assembler == nil {
		g.processPromHistograms(eventList, histograms)
	}

	if g.metricsCount {
		for _, e := range eventList {
			if _, hasLabels := e.ModuleFields["labels"]; hasLabels {
				e.RootFields["metrics_count"] = len(e.ModuleFields) - 1
			} else {
				e.RootFields["metrics_count"] = len(e.ModuleFields)
			}
		}
	}

	return eventList
}

func (g *remoteWriteTypedGenerator) readSample(metric *model.Sample) (name string, labels mapstr.M, val float64, ts time.Time, ok bool) {
	val = float64(metric.Value)
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return "", nil, 0, time.Time{}, false
	}
	ts = metric.Timestamp.Time()
	labels = mapstr.M{}
	name = string(metric.Metric["__name__"])
	for k, v := range metric.Metric {
		if k == "__name__" {
			continue
		}
		labels[string(k)] = v
	}
	return name, labels, val, ts, true
}

func (g *remoteWriteTypedGenerator) classifyBucketSamples(metrics model.Samples) []bucketSample {
	var batch []bucketSample
	for _, metric := range metrics {
		if metric == nil {
			continue
		}
		name, labels, val, ts, ok := g.readSample(metric)
		if !ok {
			continue
		}
		if g.findMetricType(name, labels) != histogramType {
			continue
		}
		labelsClone := labels.Clone()
		_ = labelsClone.Delete("le")
		le, _ := labels.GetValue("le")
		upperBound, ok := parseLEUpperBound(string(le.(model.LabelValue)))
		if !ok {
			continue
		}
		batch = append(batch, bucketSample{
			bucketMetricName: name,
			labels:           labelsClone,
			timestamp:        ts,
			buckets: []bucketUpdate{{
				upperBound:      upperBound,
				cumulativeCount: val,
			}},
		})
	}
	return batch
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

// processPromHistograms converts request-scoped Prometheus histograms to Elasticsearch histograms.
func (g *remoteWriteTypedGenerator) processPromHistograms(eventList map[string]mb.Event, histograms map[string]histogram) {
	for _, histogram := range histograms {
		labelsHash := histogram.labels.String() + histogram.timestamp.String()
		if _, ok := eventList[labelsHash]; !ok {
			eventList[labelsHash] = mb.Event{
				ModuleFields: mapstr.M{},
				Timestamp:    histogram.timestamp,
			}
			if len(histogram.labels) > 0 {
				eventList[labelsHash].ModuleFields["labels"] = histogram.labels
			}
		}

		e := eventList[labelsHash]
		hist := p.Histogram{Bucket: histogram.buckets}
		name := strings.TrimSuffix(histogram.metricName, "_bucket")
		e.ModuleFields.Update(mapstr.M{
			name: mapstr.M{
				"histogram": collector.PromHistogramToES(g.counterCache, histogram.metricName, histogram.labels, &hist),
			},
		})
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

func loadRemoteWriteModuleConfig(mod mb.Module) (config, error) {
	cfg := defaultConfig
	if err := mod.UnpackConfig(&cfg); err != nil {
		return cfg, err
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}
