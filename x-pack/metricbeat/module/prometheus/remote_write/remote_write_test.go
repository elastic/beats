// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package remote_write

import (
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
	p "github.com/elastic/beats/v8/metricbeat/helper/prometheus"
	xcollector "github.com/elastic/beats/v8/x-pack/metricbeat/module/prometheus/collector"
)

// TestGenerateEventsCounter tests counter simple cases
func TestGenerateEventsCounter(t *testing.T) {

	counters := xcollector.NewCounterCache(1 * time.Second)

	g := remoteWriteTypedGenerator{
		counterCache: counters,
		rateCounters: true,
	}
	g.counterCache.Start()
	timestamp := model.Time(424242)
	labels := common.MapStr{
		"listener_name": model.LabelValue("http"),
	}
	// first fetch
	metrics := model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(42),
			Timestamp: timestamp,
		},
	}
	events := g.GenerateEvents(metrics)

	expected := common.MapStr{
		"net_conntrack_listener_conn_closed_total": common.MapStr{
			"counter": float64(42),
			"rate":    float64(0),
		},
		"labels": labels,
	}

	assert.Equal(t, len(events), 1)
	e := events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected)

	// repeat in order to test the rate
	metrics = model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(45),
			Timestamp: timestamp,
		},
	}
	events = g.GenerateEvents(metrics)

	expected = common.MapStr{
		"net_conntrack_listener_conn_closed_total": common.MapStr{
			"counter": float64(45),
			"rate":    float64(3),
		},
		"labels": labels,
	}

	assert.Equal(t, len(events), 1)
	e = events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected)

}

// TestGenerateEventsCounterSameLabels tests multiple counters with same labels
func TestGenerateEventsCounterSameLabels(t *testing.T) {

	counters := xcollector.NewCounterCache(1 * time.Second)

	g := remoteWriteTypedGenerator{
		counterCache: counters,
		rateCounters: true,
	}
	g.counterCache.Start()
	timestamp := model.Time(424242)
	labels := common.MapStr{
		"listener_name": model.LabelValue("http"),
	}

	// first fetch
	metrics := model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(42),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(43),
			Timestamp: timestamp,
		},
	}
	events := g.GenerateEvents(metrics)

	expected := common.MapStr{
		"net_conntrack_listener_conn_closed_total": common.MapStr{
			"counter": float64(42),
			"rate":    float64(0),
		},
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(43),
			"rate":    float64(0),
		},
		"labels": labels,
	}

	assert.Equal(t, len(events), 1)
	e := events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected)

	// repeat in order to test the rate
	metrics = model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(45),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(47),
			Timestamp: timestamp,
		},
	}
	events = g.GenerateEvents(metrics)

	expected = common.MapStr{
		"net_conntrack_listener_conn_closed_total": common.MapStr{
			"counter": float64(45),
			"rate":    float64(3),
		},
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(47),
			"rate":    float64(4),
		},
		"labels": labels,
	}

	assert.Equal(t, len(events), 1)
	e = events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected)

}

// TestGenerateEventsCounterDifferentLabels tests multiple counters with different labels
func TestGenerateEventsCounterDifferentLabels(t *testing.T) {

	counters := xcollector.NewCounterCache(1 * time.Second)

	g := remoteWriteTypedGenerator{
		counterCache: counters,
		rateCounters: true,
	}
	g.counterCache.Start()

	timestamp := model.Time(424242)
	labels := common.MapStr{
		"listener_name": model.LabelValue("http"),
	}
	labels2 := common.MapStr{
		"listener_name": model.LabelValue("http"),
		"device":        model.LabelValue("eth0"),
	}

	// first fetch
	metrics := model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(42),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(43),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
				"device":        "eth0",
			},
			Value:     model.SampleValue(44),
			Timestamp: timestamp,
		},
	}
	events := g.GenerateEvents(metrics)

	expected1 := common.MapStr{
		"net_conntrack_listener_conn_closed_total": common.MapStr{
			"counter": float64(42),
			"rate":    float64(0),
		},
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(43),
			"rate":    float64(0),
		},
		"labels": labels,
	}
	expected2 := common.MapStr{
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(44),
			"rate":    float64(0),
		},
		"labels": labels2,
	}

	assert.Equal(t, len(events), 2)
	e := events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected1)
	e = events[labels2.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected2)

	// repeat in order to test the rate
	metrics = model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(45),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(47),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
				"device":        "eth0",
			},
			Value:     model.SampleValue(50),
			Timestamp: timestamp,
		},
	}
	events = g.GenerateEvents(metrics)

	expected1 = common.MapStr{
		"net_conntrack_listener_conn_closed_total": common.MapStr{
			"counter": float64(45),
			"rate":    float64(3),
		},
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(47),
			"rate":    float64(4),
		},
		"labels": labels,
	}
	expected2 = common.MapStr{
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(50),
			"rate":    float64(6),
		},
		"labels": labels2,
	}

	assert.Equal(t, len(events), 2)
	e = events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected1)
	e = events[labels2.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected2)

}

// TestGenerateEventsGaugeDifferentLabels tests multiple gauges with different labels
func TestGenerateEventsGaugeDifferentLabels(t *testing.T) {

	counters := xcollector.NewCounterCache(1 * time.Second)

	g := remoteWriteTypedGenerator{
		counterCache: counters,
		rateCounters: true,
	}
	g.counterCache.Start()
	timestamp := model.Time(424242)
	labels := common.MapStr{
		"listener_name": model.LabelValue("http"),
	}
	labels2 := common.MapStr{
		"listener_name": model.LabelValue("http"),
		"device":        model.LabelValue("eth0"),
	}

	// first fetch
	metrics := model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(42),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(43),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
				"device":        "eth0",
			},
			Value:     model.SampleValue(44),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_open",
				"listener_name": "http",
				"device":        "eth0",
			},
			Value:     model.SampleValue(49),
			Timestamp: timestamp,
		},
	}
	events := g.GenerateEvents(metrics)

	expected1 := common.MapStr{
		"net_conntrack_listener_conn_closed_total": common.MapStr{
			"counter": float64(42),
			"rate":    float64(0),
		},
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(43),
			"rate":    float64(0),
		},
		"labels": labels,
	}
	expected2 := common.MapStr{
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(44),
			"rate":    float64(0),
		},
		"net_conntrack_listener_conn_open": common.MapStr{
			"value": float64(49),
		},
		"labels": labels2,
	}

	assert.Equal(t, len(events), 2)
	e := events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected1)
	e = events[labels2.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected2)

	// repeat in order to test the rate
	metrics = model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(45),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(47),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
				"device":        "eth0",
			},
			Value:     model.SampleValue(50),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_open",
				"listener_name": "http",
				"device":        "eth0",
			},
			Value:     model.SampleValue(59),
			Timestamp: timestamp,
		},
	}
	events = g.GenerateEvents(metrics)

	expected1 = common.MapStr{
		"net_conntrack_listener_conn_closed_total": common.MapStr{
			"counter": float64(45),
			"rate":    float64(3),
		},
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(47),
			"rate":    float64(4),
		},
		"labels": labels,
	}
	expected2 = common.MapStr{
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(50),
			"rate":    float64(6),
		},
		"net_conntrack_listener_conn_open": common.MapStr{
			"value": float64(59),
		},
		"labels": labels2,
	}

	assert.Equal(t, len(events), 2)
	e = events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected1)
	e = events[labels2.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected2)

}

// TestGenerateEventsQuantilesDifferentLabels tests summaries with different labels
func TestGenerateEventsQuantilesDifferentLabels(t *testing.T) {

	counters := xcollector.NewCounterCache(1 * time.Second)

	g := remoteWriteTypedGenerator{
		counterCache: counters,
		rateCounters: true,
	}
	g.counterCache.Start()

	timestamp := model.Time(424242)
	labels := common.MapStr{
		"runtime":  model.LabelValue("linux"),
		"quantile": model.LabelValue("0.25"),
	}
	labels2 := common.MapStr{
		"runtime":  model.LabelValue("linux"),
		"quantile": model.LabelValue("0.50"),
	}
	labels3 := common.MapStr{
		"runtime": model.LabelValue("linux"),
	}

	// first fetch
	metrics := model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds",
				"runtime":  "linux",
				"quantile": "0.25",
			},
			Value:     model.SampleValue(42),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds",
				"runtime":  "linux",
				"quantile": "0.50",
			},
			Value:     model.SampleValue(43),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds_sum",
				"runtime":  "linux",
			},
			Value:     model.SampleValue(44),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds_count",
				"runtime":  "linux",
			},
			Value:     model.SampleValue(45),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds_2",
				"runtime":  "linux",
				"quantile": "0.25",
			},
			Value:     model.SampleValue(46),
			Timestamp: timestamp,
		},
	}
	events := g.GenerateEvents(metrics)

	expected := common.MapStr{
		"go_gc_duration_seconds": common.MapStr{
			"value": float64(42),
		},
		"go_gc_duration_seconds_2": common.MapStr{
			"value": float64(46),
		},
		"labels": labels,
	}
	expected2 := common.MapStr{
		"go_gc_duration_seconds": common.MapStr{
			"value": float64(43),
		},
		"labels": labels2,
	}
	expected3 := common.MapStr{
		"go_gc_duration_seconds_count": common.MapStr{
			"counter": float64(45),
			"rate":    float64(0),
		},
		"go_gc_duration_seconds_sum": common.MapStr{
			"counter": float64(44),
			"rate":    float64(0),
		},
		"labels": labels3,
	}

	assert.Equal(t, len(events), 3)
	e := events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected)
	e = events[labels2.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected2)
	e = events[labels3.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected3)

	// repeat in order to test the rate
	metrics = model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds",
				"runtime":  "linux",
				"quantile": "0.25",
			},
			Value:     model.SampleValue(52),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds",
				"runtime":  "linux",
				"quantile": "0.50",
			},
			Value:     model.SampleValue(53),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds_sum",
				"runtime":  "linux",
			},
			Value:     model.SampleValue(54),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds_count",
				"runtime":  "linux",
			},
			Value:     model.SampleValue(55),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds_2",
				"runtime":  "linux",
				"quantile": "0.25",
			},
			Value:     model.SampleValue(56),
			Timestamp: timestamp,
		},
	}
	events = g.GenerateEvents(metrics)

	expected = common.MapStr{
		"go_gc_duration_seconds": common.MapStr{
			"value": float64(52),
		},
		"go_gc_duration_seconds_2": common.MapStr{
			"value": float64(56),
		},
		"labels": labels,
	}
	expected2 = common.MapStr{
		"go_gc_duration_seconds": common.MapStr{
			"value": float64(53),
		},
		"labels": labels2,
	}
	expected3 = common.MapStr{
		"go_gc_duration_seconds_count": common.MapStr{
			"counter": float64(55),
			"rate":    float64(10),
		},
		"go_gc_duration_seconds_sum": common.MapStr{
			"counter": float64(54),
			"rate":    float64(10),
		},
		"labels": labels3,
	}

	assert.Equal(t, len(events), 3)
	e = events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected)
	e = events[labels2.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected2)
	e = events[labels3.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected3)

}

// TestGenerateEventsHistogramsDifferentLabels tests histograms with different labels
func TestGenerateEventsHistogramsDifferentLabels(t *testing.T) {

	counters := xcollector.NewCounterCache(1 * time.Second)

	g := remoteWriteTypedGenerator{
		counterCache: counters,
		rateCounters: true,
	}
	g.counterCache.Start()
	timestamp := model.Time(424242)
	labels := common.MapStr{
		"runtime": model.LabelValue("linux"),
	}
	labels2 := common.MapStr{
		"runtime": model.LabelValue("darwin"),
	}

	// first fetch
	metrics := model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_duration_seconds_bucket",
				"runtime":  "linux",
				"le":       "0.25",
			},
			Value:     model.SampleValue(42),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_duration_seconds_bucket",
				"runtime":  "linux",
				"le":       "0.50",
			},
			Value:     model.SampleValue(43),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_duration_seconds_bucket",
				"runtime":  "linux",
				"le":       "+Inf",
			},
			Value:     model.SampleValue(44),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_duration_seconds_sum",
				"runtime":  "linux",
			},
			Value:     model.SampleValue(45),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_duration_seconds_count",
				"runtime":  "linux",
			},
			Value:     model.SampleValue(46),
			Timestamp: timestamp,
		},
		// second histogram same label
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_bucket",
				"runtime":  "linux",
				"le":       "0.25",
			},
			Value:     model.SampleValue(52),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_bucket",
				"runtime":  "linux",
				"le":       "0.50",
			},
			Value:     model.SampleValue(53),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_bucket",
				"runtime":  "linux",
				"le":       "+Inf",
			},
			Value:     model.SampleValue(54),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_sum",
				"runtime":  "linux",
			},
			Value:     model.SampleValue(55),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_count",
				"runtime":  "linux",
			},
			Value:     model.SampleValue(56),
			Timestamp: timestamp,
		},
		// third histogram different label
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_bucket",
				"runtime":  "darwin",
				"le":       "0.25",
			},
			Value:     model.SampleValue(62),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_bucket",
				"runtime":  "darwin",
				"le":       "0.50",
			},
			Value:     model.SampleValue(63),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_bucket",
				"runtime":  "darwin",
				"le":       "+Inf",
			},
			Value:     model.SampleValue(64),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_sum",
				"runtime":  "darwin",
			},
			Value:     model.SampleValue(65),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_count",
				"runtime":  "darwin",
			},
			Value:     model.SampleValue(66),
			Timestamp: timestamp,
		},
	}
	events := g.GenerateEvents(metrics)

	expected := common.MapStr{
		"http_request_duration_seconds": common.MapStr{
			"histogram": common.MapStr{
				"values": []float64{float64(0.125), float64(0.375), float64(0.75)},
				"counts": []uint64{uint64(0), uint64(0), uint64(0)},
			},
		},
		"http_request_duration_seconds_sum": common.MapStr{
			"counter": float64(45),
			"rate":    float64(0),
		},
		"http_request_duration_seconds_count": common.MapStr{
			"counter": float64(46),
			"rate":    float64(0),
		},
		"http_request_bytes": common.MapStr{
			"histogram": common.MapStr{
				"values": []float64{float64(0.125), float64(0.375), float64(0.75)},
				"counts": []uint64{uint64(0), uint64(0), uint64(0)},
			},
		},
		"http_request_bytes_sum": common.MapStr{
			"counter": float64(55),
			"rate":    float64(0),
		},
		"http_request_bytes_count": common.MapStr{
			"counter": float64(56),
			"rate":    float64(0),
		},
		"labels": labels,
	}
	expected2 := common.MapStr{
		"http_request_bytes": common.MapStr{
			"histogram": common.MapStr{
				"values": []float64{float64(0.125), float64(0.375), float64(0.75)},
				"counts": []uint64{uint64(0), uint64(0), uint64(0)},
			},
		},
		"http_request_bytes_sum": common.MapStr{
			"counter": float64(65),
			"rate":    float64(0),
		},
		"http_request_bytes_count": common.MapStr{
			"counter": float64(66),
			"rate":    float64(0),
		},
		"labels": labels2,
	}

	assert.Equal(t, 2, len(events))
	e := events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected)
	e = events[labels2.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected2)

	// repeat in order to test the rate
	metrics = model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_duration_seconds_bucket",
				"runtime":  "linux",
				"le":       "0.25",
			},
			Value:     model.SampleValue(142),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_duration_seconds_bucket",
				"runtime":  "linux",
				"le":       "0.50",
			},
			Value:     model.SampleValue(143),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_duration_seconds_bucket",
				"runtime":  "linux",
				"le":       "+Inf",
			},
			Value:     model.SampleValue(144),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_duration_seconds_sum",
				"runtime":  "linux",
			},
			Value:     model.SampleValue(145),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_duration_seconds_count",
				"runtime":  "linux",
			},
			Value:     model.SampleValue(146),
			Timestamp: timestamp,
		},
		// second histogram same label
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_bucket",
				"runtime":  "linux",
				"le":       "0.25",
			},
			Value:     model.SampleValue(252),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_bucket",
				"runtime":  "linux",
				"le":       "0.50",
			},
			Value:     model.SampleValue(253),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_bucket",
				"runtime":  "linux",
				"le":       "+Inf",
			},
			Value:     model.SampleValue(254),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_sum",
				"runtime":  "linux",
			},
			Value:     model.SampleValue(255),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_count",
				"runtime":  "linux",
			},
			Value:     model.SampleValue(256),
			Timestamp: timestamp,
		},
		// third histogram different label
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_bucket",
				"runtime":  "darwin",
				"le":       "0.25",
			},
			Value:     model.SampleValue(362),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_bucket",
				"runtime":  "darwin",
				"le":       "0.50",
			},
			Value:     model.SampleValue(363),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_bucket",
				"runtime":  "darwin",
				"le":       "+Inf",
			},
			Value:     model.SampleValue(364),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_sum",
				"runtime":  "darwin",
			},
			Value:     model.SampleValue(365),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_bytes_count",
				"runtime":  "darwin",
			},
			Value:     model.SampleValue(366),
			Timestamp: timestamp,
		},
	}
	events = g.GenerateEvents(metrics)

	expected = common.MapStr{
		"http_request_duration_seconds": common.MapStr{
			"histogram": common.MapStr{
				"values": []float64{float64(0.125), float64(0.375), float64(0.75)},
				"counts": []uint64{uint64(100), uint64(0), uint64(0)},
			},
		},
		"http_request_duration_seconds_sum": common.MapStr{
			"counter": float64(145),
			"rate":    float64(100),
		},
		"http_request_duration_seconds_count": common.MapStr{
			"counter": float64(146),
			"rate":    float64(100),
		},
		"http_request_bytes": common.MapStr{
			"histogram": common.MapStr{
				"values": []float64{float64(0.125), float64(0.375), float64(0.75)},
				"counts": []uint64{uint64(200), uint64(0), uint64(0)},
			},
		},
		"http_request_bytes_sum": common.MapStr{
			"counter": float64(255),
			"rate":    float64(200),
		},
		"http_request_bytes_count": common.MapStr{
			"counter": float64(256),
			"rate":    float64(200),
		},
		"labels": labels,
	}
	expected2 = common.MapStr{
		"http_request_bytes": common.MapStr{
			"histogram": common.MapStr{
				"values": []float64{float64(0.125), float64(0.375), float64(0.75)},
				"counts": []uint64{uint64(300), uint64(0), uint64(0)},
			},
		},
		"http_request_bytes_sum": common.MapStr{
			"counter": float64(365),
			"rate":    float64(300),
		},
		"http_request_bytes_count": common.MapStr{
			"counter": float64(366),
			"rate":    float64(300),
		},
		"labels": labels2,
	}

	assert.Equal(t, 2, len(events))
	e = events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected)
	e = events[labels2.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected2)
}

// TestGenerateEventsCounterWithDefinedPattern tests counter with defined pattern
func TestGenerateEventsCounterWithDefinedPattern(t *testing.T) {

	counters := xcollector.NewCounterCache(1 * time.Second)

	counterPatterns, err := p.CompilePatternList(&[]string{"_mycounter"})
	if err != nil {
		panic(err)
	}
	g := remoteWriteTypedGenerator{
		counterCache:    counters,
		rateCounters:    true,
		counterPatterns: counterPatterns,
	}

	g.counterCache.Start()

	timestamp := model.Time(424242)
	labels := common.MapStr{
		"listener_name": model.LabelValue("http"),
	}

	// first fetch
	metrics := model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_mycounter",
				"listener_name": "http",
			},
			Value:     model.SampleValue(42),
			Timestamp: timestamp,
		},
	}
	events := g.GenerateEvents(metrics)

	expected := common.MapStr{
		"net_conntrack_listener_conn_closed_mycounter": common.MapStr{
			"counter": float64(42),
			"rate":    float64(0),
		},
		"labels": labels,
	}

	assert.Equal(t, len(events), 1)
	e := events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected)

	// repeat in order to test the rate
	metrics = model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_mycounter",
				"listener_name": "http",
			},
			Value:     model.SampleValue(45),
			Timestamp: timestamp,
		},
	}
	events = g.GenerateEvents(metrics)

	expected = common.MapStr{
		"net_conntrack_listener_conn_closed_mycounter": common.MapStr{
			"counter": float64(45),
			"rate":    float64(3),
		},
		"labels": labels,
	}

	assert.Equal(t, len(events), 1)
	e = events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected)

}

// TestGenerateEventsHistogramWithDefinedPattern tests histogram with defined pattern
func TestGenerateEventsHistogramWithDefinedPattern(t *testing.T) {

	counters := xcollector.NewCounterCache(1 * time.Second)

	histogramPatterns, err := p.CompilePatternList(&[]string{"_myhistogram"})
	if err != nil {
		panic(err)
	}
	g := remoteWriteTypedGenerator{
		counterCache:      counters,
		rateCounters:      true,
		histogramPatterns: histogramPatterns,
	}

	g.counterCache.Start()
	timestamp := model.Time(424242)
	labels := common.MapStr{
		"listener_name": model.LabelValue("http"),
	}

	// first fetch
	metrics := model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_myhistogram",
				"listener_name": "http",
				"le":            "20",
			},
			Value:     model.SampleValue(42),
			Timestamp: timestamp,
		},
	}
	events := g.GenerateEvents(metrics)

	expected := common.MapStr{
		"net_conntrack_listener_conn_closed_myhistogram": common.MapStr{
			"histogram": common.MapStr{
				"values": []float64{float64(10)},
				"counts": []uint64{uint64(0)},
			},
		},
		"labels": labels,
	}

	assert.Equal(t, len(events), 1)
	e := events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected)

	// repeat in order to test the rate
	metrics = model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_myhistogram",
				"listener_name": "http",
				"le":            "20",
			},
			Value:     model.SampleValue(45),
			Timestamp: timestamp,
		},
	}
	events = g.GenerateEvents(metrics)

	expected = common.MapStr{
		"net_conntrack_listener_conn_closed_myhistogram": common.MapStr{
			"histogram": common.MapStr{
				"values": []float64{float64(10)},
				"counts": []uint64{uint64(3)},
			},
		},
		"labels": labels,
	}

	assert.Equal(t, len(events), 1)
	e = events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected)

}
