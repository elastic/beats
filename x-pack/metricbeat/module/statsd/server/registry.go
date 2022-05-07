// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"time"

	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/helper/labelhash"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var logger = logp.NewLogger("statd")

type metric struct {
	name       string
	tags       map[string]string
	lastSeen   time.Time
	sampleRate float32
	metric     interface{}
}

type registry struct {
	metrics    map[string]map[string]*metric
	ttl        time.Duration
	lastReport time.Time
}

type setMetric struct {
	set map[string]struct{}
}

func (s *setMetric) Add(val string) {
	s.set[val] = struct{}{}
}

func (s *setMetric) Reset() {
	s.set = map[string]struct{}{}
}

func (s *setMetric) Count() int {
	return len(s.set)
}

func newSetMetric() *setMetric {
	s := setMetric{}
	s.Reset()
	return &s
}

type deltaGaugeMetric struct {
	value float64
}

func (d *deltaGaugeMetric) Inc(val float64) {
	d.value += val
}

func (d *deltaGaugeMetric) Set(val float64) {
	d.value = val
}

func (d *deltaGaugeMetric) Value() float64 {
	return d.value
}

// SamplingTimer is a timer that supports sampling
type samplingTimer struct {
	metrics.Timer
	meter     metrics.Meter
	histogram metrics.Histogram
}

// NewSamplingTimer returns a new SamplingTimer
func newSamplingTimer() *samplingTimer {
	m := metrics.NewMeter()
	h := metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015))

	return &samplingTimer{
		Timer:     metrics.NewCustomTimer(h, m),
		meter:     m,
		histogram: h,
	}
}

// SampledUpdate will update the timer a sampled measurement
func (s *samplingTimer) SampledUpdate(d time.Duration, sampleRate float64) {
	s.histogram.Update(int64(d))
	s.meter.Mark(int64(1 / sampleRate))
}

// Snapshot gets a snapshot of the SamplingTimer
func (s *samplingTimer) Snapshot() samplingTimerSnapshot {
	return samplingTimerSnapshot{
		histogram: s.histogram.Snapshot(),
		meter:     s.meter.Snapshot(),
	}
}

type samplingTimerSnapshot struct {
	histogram metrics.Histogram
	meter     metrics.Meter
}

// Count returns the number of events recorded at the time the snapshot was
// taken.
func (t *samplingTimerSnapshot) Count() int64 { return t.meter.Count() }

// Max returns the maximum value at the time the snapshot was taken.
func (t *samplingTimerSnapshot) Max() int64 { return t.histogram.Max() }

// Mean returns the mean value at the time the snapshot was taken.
func (t *samplingTimerSnapshot) Mean() float64 { return t.histogram.Mean() }

// Min returns the minimum value at the time the snapshot was taken.
func (t *samplingTimerSnapshot) Min() int64 { return t.histogram.Min() }

// Percentile returns an arbitrary percentile of sampled values at the time the
// snapshot was taken.
func (t *samplingTimerSnapshot) Percentile(p float64) float64 {
	return t.histogram.Percentile(p)
}

// Percentiles returns a slice of arbitrary percentiles of sampled values at
// the time the snapshot was taken.
func (t *samplingTimerSnapshot) Percentiles(ps []float64) []float64 {
	return t.histogram.Percentiles(ps)
}

// Rate1 returns the one-minute moving average rate of events per second at the
// time the snapshot was taken.
func (t *samplingTimerSnapshot) Rate1() float64 { return t.meter.Rate1() }

// Rate5 returns the five-minute moving average rate of events per second at
// the time the snapshot was taken.
func (t *samplingTimerSnapshot) Rate5() float64 { return t.meter.Rate5() }

// Rate15 returns the fifteen-minute moving average rate of events per second
// at the time the snapshot was taken.
func (t *samplingTimerSnapshot) Rate15() float64 { return t.meter.Rate15() }

// RateMean returns the meter's mean rate of events per second at the time the
// snapshot was taken.
func (t *samplingTimerSnapshot) RateMean() float64 { return t.meter.RateMean() }

// Snapshot returns the snapshot.
func (t *samplingTimerSnapshot) Snapshot() metrics.Timer { return t }

// StdDev returns the standard deviation of the values at the time the snapshot
// was taken.
func (t *samplingTimerSnapshot) StdDev() float64 { return t.histogram.StdDev() }

// Stop is a no-op.
func (t *samplingTimerSnapshot) Stop() {}

// Sum returns the sum at the time the snapshot was taken.
func (t *samplingTimerSnapshot) Sum() int64 { return t.histogram.Sum() }

// Time panics.
func (*samplingTimerSnapshot) Time(func()) {
	panic("Time called on a samplingTimerSnapshot")
}

// Update panics.
func (*samplingTimerSnapshot) Update(time.Duration) {
	panic("Update called on a samplingTimerSnapshot")
}

// Record the duration of an event that started at a time and ends now.
func (t *samplingTimerSnapshot) UpdateSince(ts time.Time) {
	panic("Update called on a samplingTimerSnapshot")
}

// Variance returns the variance of the values in the sample.
func (t *samplingTimerSnapshot) Variance() float64 {
	return t.histogram.Variance()
}

type metricsGroup struct {
	tags    map[string]string
	metrics mapstr.M
}

func (r *registry) getMetric(metric interface{}) map[string]interface{} {
	values := map[string]interface{}{}
	switch m := metric.(type) {
	case metrics.Counter:
		values["count"] = m.Count()
		m.Clear()
	case *deltaGaugeMetric:
		values["value"] = m.Value()
	case metrics.Histogram:
		h := m.Snapshot()
		ps := h.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
		values["count"] = h.Count()
		values["min"] = h.Min()
		values["max"] = h.Max()
		values["mean"] = h.Mean()
		values["stddev"] = h.StdDev()
		values["median"] = ps[0]
		values["p75"] = ps[1]
		values["p95"] = ps[2]
		values["p99"] = ps[3]
		values["p99_9"] = ps[4]
	case *samplingTimer:
		t := m.Snapshot()
		ps := t.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
		values["count"] = t.Count()
		values["min"] = t.Min()
		values["max"] = t.Max()
		values["mean"] = t.Mean()
		values["stddev"] = t.StdDev()
		values["median"] = ps[0]
		values["p75"] = ps[1]
		values["p95"] = ps[2]
		values["p99"] = ps[3]
		values["p99_9"] = ps[4]
		values["1m_rate"] = t.Rate1()
		values["5m_rate"] = t.Rate5()
		values["15m_rate"] = t.Rate15()
		values["mean_rate"] = t.RateMean()
	case *setMetric:
		values["count"] = m.Count()
		m.Reset()
	}
	return values
}

func (r *registry) GetAll() []metricsGroup {
	var tags map[string]string
	now := time.Now()
	cutOff := now.Add(-r.ttl)

	// we do this to ensure metrics are reported at least once
	if cutOff.After(r.lastReport) {
		cutOff = r.lastReport
	}

	tagGroups := []metricsGroup{}
	for tagGroupKey, metricsMap := range r.metrics {
		fields := mapstr.M{}
		for key, m := range metricsMap {

			// cleanups according to ttl
			if r.ttl > 0 && m.lastSeen.Before(cutOff) {
				if stoppable, ok := m.metric.(metrics.Stoppable); ok {
					stoppable.Stop()
				}
				delete(metricsMap, key)
				continue
			}

			// all the .tags are the same for this metricsMap
			// we just need one
			tags = m.tags
			fields[m.name] = r.getMetric(m.metric)
		}

		// cleanup the tag group if it's empty
		if len(metricsMap) == 0 {
			delete(r.metrics, tagGroupKey)
			continue
		}

		tagGroups = append(tagGroups, metricsGroup{
			metrics: fields,
			tags:    tags,
		})

	}
	r.lastReport = now
	return tagGroups
}

func (r *registry) Delete(name string, tags map[string]string) {
	if group, ok := r.metrics[r.metricHash(tags)]; ok {
		delete(group, name)
	}
}

func (r *registry) getOrNew(name string, tags map[string]string, new func() interface{}) interface{} {
	tagsKey := r.metricHash(tags)
	tc, ok := r.metrics[tagsKey]
	if !ok {
		counter := new()
		r.metrics[tagsKey] = map[string]*metric{name: &metric{
			metric:   counter,
			name:     name,
			tags:     tags,
			lastSeen: time.Now(),
		}}
		return counter
	}

	c, ok := tc[name]
	if !ok {
		counter := new()
		tc[name] = &metric{
			metric:   counter,
			name:     name,
			tags:     tags,
			lastSeen: time.Now(),
		}
		return counter
	}

	c.lastSeen = time.Now()

	return c.metric
}

func (r *registry) clearTypeChanged(name string, tags map[string]string) {
	// type was changed
	// we can try to support the situation where a new version of the app has changed a type in
	// a metric by deleting the old one and creating a new one
	logger.With("name", name).Warn("metric changed type")
	r.Delete(name, tags)
}

func (r *registry) GetOrNewCounter(name string, tags map[string]string) metrics.Counter {
	maybeCounter := r.getOrNew(name, tags, func() interface{} { return metrics.NewCounter() })
	counter, ok := maybeCounter.(metrics.Counter)
	if ok {
		return counter
	}

	r.clearTypeChanged(name, tags)
	return r.GetOrNewCounter(name, tags)

}

func (r *registry) GetOrNewTimer(name string, tags map[string]string) *samplingTimer {
	timer, ok := r.getOrNew(name, tags, func() interface{} { return newSamplingTimer() }).(*samplingTimer)
	if ok {
		return timer
	}

	r.clearTypeChanged(name, tags)
	return r.GetOrNewTimer(name, tags)
}

func (r *registry) GetOrNewGauge64(name string, tags map[string]string) *deltaGaugeMetric {
	gauge, ok := r.getOrNew(name, tags, func() interface{} { return &deltaGaugeMetric{} }).(*deltaGaugeMetric)
	if ok {
		return gauge
	}

	r.clearTypeChanged(name, tags)
	return r.GetOrNewGauge64(name, tags)
}

func (r *registry) GetOrNewHistogram(name string, tags map[string]string) metrics.Histogram {
	histogram, ok := r.getOrNew(name, tags, func() interface{} { return metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015)) }).(metrics.Histogram)
	if ok {
		return histogram
	}

	r.clearTypeChanged(name, tags)
	return r.GetOrNewHistogram(name, tags)
}

func (r *registry) GetOrNewSet(name string, tags map[string]string) *setMetric {
	setmetric, ok := r.getOrNew(name, tags, func() interface{} { return newSetMetric() }).(*setMetric)
	if ok {
		return setmetric
	}

	r.clearTypeChanged(name, tags)
	return r.GetOrNewSet(name, tags)
}

func (r *registry) metricHash(tags map[string]string) string {
	mapstrTags := mapstr.M{}
	for k, v := range tags {
		mapstrTags[k] = v
	}
	return labelhash.LabelHash(mapstrTags)
}
