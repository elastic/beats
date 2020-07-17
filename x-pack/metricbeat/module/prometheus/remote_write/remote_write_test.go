// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package remote_write

import (
	"github.com/prometheus/common/model"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	xcollector "github.com/elastic/beats/v7/x-pack/metricbeat/module/prometheus/collector"
)

// TestGenerateEventsCounter tests counter simple cases
func TestGenerateEventsCounter(t *testing.T) {

	counters := xcollector.NewCounterCache(1 * time.Second)

	g := RemoteWriteTypedGenerator{
		CounterCache: counters,
		RateCounters: true,
	}
	g.CounterCache.Start()
	labels := common.MapStr{
		"listener_name": model.LabelValue("http"),
	}

	// first fetch
	metrics := model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value: model.SampleValue(42),
			Timestamp: model.Time(424242),
		},
	}
	events := g.GenerateEvents(metrics)


	expected := common.MapStr{
			"net_conntrack_listener_conn_closed_total": common.MapStr{
				"counter": float64(42),
				"rate": float64(0),
			},
			"labels": labels,
	}

	assert.Equal(t, len(events), 1)
	e := events[labels.String()]
	assert.EqualValues(t, e.ModuleFields, expected)


	// repeat in order to test the rate
	metrics = model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value: model.SampleValue(45),
			Timestamp: model.Time(424242),
		},
	}
	events = g.GenerateEvents(metrics)

	expected = common.MapStr{
		"net_conntrack_listener_conn_closed_total": common.MapStr{
			"counter": float64(45),
			"rate": float64(3),
		},
		"labels": labels,
	}

	assert.Equal(t, len(events), 1)
	e = events[labels.String()]
	assert.EqualValues(t, e.ModuleFields, expected)

}


// TestGenerateEventsCounterSameLabels tests multiple counters with same labels
func TestGenerateEventsCounterSameLabels(t *testing.T) {

	counters := xcollector.NewCounterCache(1 * time.Second)

	g := RemoteWriteTypedGenerator{
		CounterCache: counters,
		RateCounters: true,
	}
	g.CounterCache.Start()
	labels := common.MapStr{
		"listener_name": model.LabelValue("http"),
	}

	// first fetch
	metrics := model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value: model.SampleValue(42),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
			},
			Value: model.SampleValue(43),
			Timestamp: model.Time(424242),
		},
	}
	events := g.GenerateEvents(metrics)


	expected := common.MapStr{
		"net_conntrack_listener_conn_closed_total": common.MapStr{
			"counter": float64(42),
			"rate": float64(0),
		},
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(43),
			"rate": float64(0),
		},
		"labels": labels,
	}

	assert.Equal(t, len(events), 1)
	e := events[labels.String()]
	assert.EqualValues(t, e.ModuleFields, expected)


	// repeat in order to test the rate
	metrics = model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value: model.SampleValue(45),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
			},
			Value: model.SampleValue(47),
			Timestamp: model.Time(424242),
		},
	}
	events = g.GenerateEvents(metrics)

	expected = common.MapStr{
		"net_conntrack_listener_conn_closed_total": common.MapStr{
			"counter": float64(45),
			"rate": float64(3),
		},
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(47),
			"rate": float64(4),
		},
		"labels": labels,
	}

	assert.Equal(t, len(events), 1)
	e = events[labels.String()]
	assert.EqualValues(t, e.ModuleFields, expected)

}


// TestGenerateEventsCounterDifferentLabels tests multiple counters with different labels
func TestGenerateEventsCounterDifferentLabels(t *testing.T) {

	counters := xcollector.NewCounterCache(1 * time.Second)

	g := RemoteWriteTypedGenerator{
		CounterCache: counters,
		RateCounters: true,
	}
	g.CounterCache.Start()
	labels := common.MapStr{
		"listener_name": model.LabelValue("http"),
	}
	labels2 := common.MapStr{
		"listener_name": model.LabelValue("http"),
		"device": model.LabelValue("eth0"),
	}

	// first fetch
	metrics := model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value: model.SampleValue(42),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
			},
			Value: model.SampleValue(43),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
				"device": "eth0",
			},
			Value: model.SampleValue(44),
			Timestamp: model.Time(424242),
		},
	}
	events := g.GenerateEvents(metrics)


	expected1 := common.MapStr{
		"net_conntrack_listener_conn_closed_total": common.MapStr{
			"counter": float64(42),
			"rate": float64(0),
		},
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(43),
			"rate": float64(0),
		},
		"labels": labels,
	}
	expected2 := common.MapStr{
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(44),
			"rate": float64(0),
		},
		"labels": labels2,
	}

	assert.Equal(t, len(events), 2)
	e := events[labels.String()]
	assert.EqualValues(t, e.ModuleFields, expected1)
	e = events[labels2.String()]
	assert.EqualValues(t, e.ModuleFields, expected2)


	// repeat in order to test the rate
	metrics = model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value: model.SampleValue(45),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
			},
			Value: model.SampleValue(47),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
				"device": "eth0",
			},
			Value: model.SampleValue(50),
			Timestamp: model.Time(424242),
		},
	}
	events = g.GenerateEvents(metrics)

	expected1 = common.MapStr{
		"net_conntrack_listener_conn_closed_total": common.MapStr{
			"counter": float64(45),
			"rate": float64(3),
		},
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(47),
			"rate": float64(4),
		},
		"labels": labels,
	}
	expected2 = common.MapStr{
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(50),
			"rate": float64(6),
		},
		"labels": labels2,
	}

	assert.Equal(t, len(events), 2)
	e = events[labels.String()]
	assert.EqualValues(t, e.ModuleFields, expected1)
	e = events[labels2.String()]
	assert.EqualValues(t, e.ModuleFields, expected2)

}

// TestGenerateEventsGaugeDifferentLabels tests multiple gauges with different labels
func TestGenerateEventsGaugeDifferentLabels(t *testing.T) {

	counters := xcollector.NewCounterCache(1 * time.Second)

	g := RemoteWriteTypedGenerator{
		CounterCache: counters,
		RateCounters: true,
	}
	g.CounterCache.Start()
	labels := common.MapStr{
		"listener_name": model.LabelValue("http"),
	}
	labels2 := common.MapStr{
		"listener_name": model.LabelValue("http"),
		"device": model.LabelValue("eth0"),
	}

	// first fetch
	metrics := model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value: model.SampleValue(42),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
			},
			Value: model.SampleValue(43),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
				"device": "eth0",
			},
			Value: model.SampleValue(44),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_open",
				"listener_name": "http",
				"device": "eth0",
			},
			Value: model.SampleValue(49),
			Timestamp: model.Time(424242),
		},
	}
	events := g.GenerateEvents(metrics)


	expected1 := common.MapStr{
		"net_conntrack_listener_conn_closed_total": common.MapStr{
			"counter": float64(42),
			"rate": float64(0),
		},
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(43),
			"rate": float64(0),
		},
		"labels": labels,
	}
	expected2 := common.MapStr{
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(44),
			"rate": float64(0),
		},
		"net_conntrack_listener_conn_open": common.MapStr{
			"value": float64(49),
		},
		"labels": labels2,
	}

	assert.Equal(t, len(events), 2)
	e := events[labels.String()]
	assert.EqualValues(t, e.ModuleFields, expected1)
	e = events[labels2.String()]
	assert.EqualValues(t, e.ModuleFields, expected2)


	// repeat in order to test the rate
	metrics = model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value: model.SampleValue(45),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
			},
			Value: model.SampleValue(47),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_panic_total",
				"listener_name": "http",
				"device": "eth0",
			},
			Value: model.SampleValue(50),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "net_conntrack_listener_conn_open",
				"listener_name": "http",
				"device": "eth0",
			},
			Value: model.SampleValue(59),
			Timestamp: model.Time(424242),
		},
	}
	events = g.GenerateEvents(metrics)

	expected1 = common.MapStr{
		"net_conntrack_listener_conn_closed_total": common.MapStr{
			"counter": float64(45),
			"rate": float64(3),
		},
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(47),
			"rate": float64(4),
		},
		"labels": labels,
	}
	expected2 = common.MapStr{
		"net_conntrack_listener_conn_panic_total": common.MapStr{
			"counter": float64(50),
			"rate": float64(6),
		},
		"net_conntrack_listener_conn_open": common.MapStr{
			"value": float64(59),
		},
		"labels": labels2,
	}

	assert.Equal(t, len(events), 2)
	e = events[labels.String()]
	assert.EqualValues(t, e.ModuleFields, expected1)
	e = events[labels2.String()]
	assert.EqualValues(t, e.ModuleFields, expected2)

}

// TestGenerateEventsQuantilesDifferentLabels tests multiple gauges with different labels
func TestGenerateEventsQuantilesDifferentLabels(t *testing.T) {

	counters := xcollector.NewCounterCache(1 * time.Second)

	g := RemoteWriteTypedGenerator{
		CounterCache: counters,
		RateCounters: true,
	}
	g.CounterCache.Start()
	labels := common.MapStr{
		"runtime": model.LabelValue("linux"),
		"quantile": model.LabelValue("0.25"),
	}
	labels2 := common.MapStr{
		"runtime": model.LabelValue("linux"),
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
				"runtime": "linux",
				"quantile": "0.25",
			},
			Value: model.SampleValue(42),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds",
				"runtime": "linux",
				"quantile": "0.50",
			},
			Value: model.SampleValue(43),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds_sum",
				"runtime": "linux",
			},
			Value: model.SampleValue(44),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds_count",
				"runtime": "linux",
			},
			Value: model.SampleValue(45),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds_2",
				"runtime": "linux",
				"quantile": "0.25",
			},
			Value: model.SampleValue(46),
			Timestamp: model.Time(424242),
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
			"counter": uint64(45),
			"rate": uint64(0),
		},
		"go_gc_duration_seconds_sum": common.MapStr{
			"counter": float64(44),
			"rate": float64(0),
		},
		"labels": labels3,
	}

	assert.Equal(t, len(events), 3)
	e := events[labels.String()]
	assert.EqualValues(t, e.ModuleFields, expected)
	e = events[labels2.String()]
	assert.EqualValues(t, e.ModuleFields, expected2)
	e = events[labels3.String()]
	assert.EqualValues(t, e.ModuleFields, expected3)


	// repeat in order to test the rate
	metrics = model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds",
				"runtime": "linux",
				"quantile": "0.25",
			},
			Value: model.SampleValue(52),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds",
				"runtime": "linux",
				"quantile": "0.50",
			},
			Value: model.SampleValue(53),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds_sum",
				"runtime": "linux",
			},
			Value: model.SampleValue(54),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds_count",
				"runtime": "linux",
			},
			Value: model.SampleValue(55),
			Timestamp: model.Time(424242),
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "go_gc_duration_seconds_2",
				"runtime": "linux",
				"quantile": "0.25",
			},
			Value: model.SampleValue(56),
			Timestamp: model.Time(424242),
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
			"counter": uint64(55),
			"rate": uint64(10),
		},
		"go_gc_duration_seconds_sum": common.MapStr{
			"counter": float64(54),
			"rate": float64(10),
		},
		"labels": labels3,
	}

	assert.Equal(t, len(events), 3)
	e = events[labels.String()]
	assert.EqualValues(t, e.ModuleFields, expected)
	e = events[labels2.String()]
	assert.EqualValues(t, e.ModuleFields, expected2)
	e = events[labels3.String()]
	assert.EqualValues(t, e.ModuleFields, expected3)

}
