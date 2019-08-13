// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"encoding/json"
	"time"

	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
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
	metrics       map[string]map[string]*metric
	reservoirSize int
	ttl           time.Duration
	lastReport    time.Time
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

type metricsGroup struct {
	tags    map[string]string
	metrics common.MapStr
}

func (r *registry) getMetric(metric interface{}) map[string]interface{} {
	values := map[string]interface{}{}
	switch m := metric.(type) {
	case metrics.Counter:
		values["count"] = m.Count()
	case metrics.GaugeFloat64:
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
	case metrics.Timer:
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
		fields := common.MapStr{}
		for key, m := range metricsMap {

			// cleanups according to ttl
			if r.ttl > 0 && m.lastSeen.Before(cutOff) {
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

func (r *registry) GetOrNewCounter(name string, tags map[string]string) metrics.Counter {
	maybeCounter := r.getOrNew(name, tags, func() interface{} { return metrics.NewCounter() })
	counter, ok := maybeCounter.(metrics.Counter)
	if ok {
		return counter
	}

	// type was changed
	// we can try to support the situation where a new version of the app has changed a type in
	// a metric by deleting the old one and creating a new one
	logger.With("name", name).Warn("metric changed type")
	r.Delete(name, tags)
	return r.GetOrNewCounter(name, tags)

}

func (r *registry) GetOrNewTimer(name string, tags map[string]string) metrics.Timer {
	timer, ok := r.getOrNew(name, tags, func() interface{} { return metrics.NewTimer() }).(metrics.Timer)
	if ok {
		return timer
	}
	// type was changed
	// we can try to support the situation where a new version of the app has changed a type in
	// a metric by deleting the old one and creating a new one
	logger.With("name", name).Warn("metric changed type")
	r.Delete(name, tags)
	return r.GetOrNewTimer(name, tags)
}

func (r *registry) GetOrNewGauge64(name string, tags map[string]string) metrics.GaugeFloat64 {
	gauge, ok := r.getOrNew(name, tags, func() interface{} { return metrics.NewGaugeFloat64() }).(metrics.GaugeFloat64)
	if ok {
		return gauge
	}
	// type was changed
	// we can try to support the situation where a new version of the app has changed a type in
	// a metric by deleting the old one and creating a new one
	logger.With("name", name).Warn("metric changed type")
	r.Delete(name, tags)
	return r.GetOrNewGauge64(name, tags)
}

func (r *registry) GetOrNewHistogram(name string, tags map[string]string) metrics.Histogram {
	histogram, ok := r.getOrNew(name, tags, func() interface{} { return metrics.NewHistogram(metrics.NewUniformSample(r.reservoirSize)) }).(metrics.Histogram)
	if ok {
		return histogram
	}
	// type was changed
	// we can try to support the situation where a new version of the app has changed a type in
	// a metric by deleting the old one and creating a new one
	logger.With("name", name).Warn("metric changed type")
	r.Delete(name, tags)
	return r.GetOrNewHistogram(name, tags)
}

func (r *registry) GetOrNewSet(name string, tags map[string]string) *setMetric {
	setmetric, ok := r.getOrNew(name, tags, func() interface{} { return newSetMetric() }).(*setMetric)
	if ok {
		return setmetric
	}
	// type was changed
	// we can try to support the situation where a new version of the app has changed a type in
	// a metric by deleting the old one and creating a new one
	logger.With("name", name).Warn("metric changed type")
	r.Delete(name, tags)
	return r.GetOrNewSet(name, tags)
}

func (r *registry) metricHash(tags map[string]string) string {
	b, err := json.Marshal(tags)
	if err != nil { // shouldn't happen on a map[string]string
		panic(err)
	}
	return string(b)
}
