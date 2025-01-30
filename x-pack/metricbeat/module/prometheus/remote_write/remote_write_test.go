// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package remote_write

import (
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"

	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	xcollector "github.com/elastic/beats/v7/x-pack/metricbeat/module/prometheus/collector"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func BenchmarkGenerateEvents(b *testing.B) {
	// Create a sample set of metrics
	metrics := createSampleMetrics()

	// Create an instance of remoteWriteTypedGenerator
	generator := remoteWriteTypedGenerator{
		// Initialize with appropriate values
		metricsCount: true,
		// Add other necessary fields
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator.GenerateEvents(metrics)
	}
}

func createSampleMetrics() model.Samples {
	now := model.TimeFromUnix(time.Now().Unix())
	return model.Samples{
		&model.Sample{
			Metric: model.Metric{
				"__name__": "http_requests_total",
				"method":   "GET",
				"status":   "200",
			},
			Value:     1234,
			Timestamp: now,
		},
		&model.Sample{
			Metric: model.Metric{
				"__name__": "http_request_duration_seconds",
				"method":   "POST",
				"path":     "/api/v1/users",
			},
			Value:     0.543,
			Timestamp: now,
		},
		&model.Sample{
			Metric: model.Metric{
				"__name__": "node_cpu_seconds_total",
				"cpu":      "0",
				"mode":     "idle",
			},
			Value:     3600.5,
			Timestamp: now,
		},
		&model.Sample{
			Metric: model.Metric{
				"__name__": "go_goroutines",
			},
			Value:     42,
			Timestamp: now,
		},
		&model.Sample{
			Metric: model.Metric{
				"__name__": "process_resident_memory_bytes",
			},
			Value:     2.5e+7,
			Timestamp: now,
		},
	}
}

// TestGenerateEventsCounter tests counter simple cases
func TestGenerateEventsCounter(t *testing.T) {

	counters := xcollector.NewCounterCache(1 * time.Second)

	g := remoteWriteTypedGenerator{
		counterCache: counters,
		rateCounters: true,
	}
	g.counterCache.Start()
	timestamp := model.Time(424242)
	labels := mapstr.M{
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

	expected := mapstr.M{
		"net_conntrack_listener_conn_closed_total": mapstr.M{
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

	expected = mapstr.M{
		"net_conntrack_listener_conn_closed_total": mapstr.M{
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
	labels := mapstr.M{
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

	expected := mapstr.M{
		"net_conntrack_listener_conn_closed_total": mapstr.M{
			"counter": float64(42),
			"rate":    float64(0),
		},
		"net_conntrack_listener_conn_panic_total": mapstr.M{
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

	expected = mapstr.M{
		"net_conntrack_listener_conn_closed_total": mapstr.M{
			"counter": float64(45),
			"rate":    float64(3),
		},
		"net_conntrack_listener_conn_panic_total": mapstr.M{
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
	labels := mapstr.M{
		"listener_name": model.LabelValue("http"),
	}
	labels2 := mapstr.M{
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

	expected1 := mapstr.M{
		"net_conntrack_listener_conn_closed_total": mapstr.M{
			"counter": float64(42),
			"rate":    float64(0),
		},
		"net_conntrack_listener_conn_panic_total": mapstr.M{
			"counter": float64(43),
			"rate":    float64(0),
		},
		"labels": labels,
	}
	expected2 := mapstr.M{
		"net_conntrack_listener_conn_panic_total": mapstr.M{
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

	expected1 = mapstr.M{
		"net_conntrack_listener_conn_closed_total": mapstr.M{
			"counter": float64(45),
			"rate":    float64(3),
		},
		"net_conntrack_listener_conn_panic_total": mapstr.M{
			"counter": float64(47),
			"rate":    float64(4),
		},
		"labels": labels,
	}
	expected2 = mapstr.M{
		"net_conntrack_listener_conn_panic_total": mapstr.M{
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
	labels := mapstr.M{
		"listener_name": model.LabelValue("http"),
	}
	labels2 := mapstr.M{
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

	expected1 := mapstr.M{
		"net_conntrack_listener_conn_closed_total": mapstr.M{
			"counter": float64(42),
			"rate":    float64(0),
		},
		"net_conntrack_listener_conn_panic_total": mapstr.M{
			"counter": float64(43),
			"rate":    float64(0),
		},
		"labels": labels,
	}
	expected2 := mapstr.M{
		"net_conntrack_listener_conn_panic_total": mapstr.M{
			"counter": float64(44),
			"rate":    float64(0),
		},
		"net_conntrack_listener_conn_open": mapstr.M{
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

	expected1 = mapstr.M{
		"net_conntrack_listener_conn_closed_total": mapstr.M{
			"counter": float64(45),
			"rate":    float64(3),
		},
		"net_conntrack_listener_conn_panic_total": mapstr.M{
			"counter": float64(47),
			"rate":    float64(4),
		},
		"labels": labels,
	}
	expected2 = mapstr.M{
		"net_conntrack_listener_conn_panic_total": mapstr.M{
			"counter": float64(50),
			"rate":    float64(6),
		},
		"net_conntrack_listener_conn_open": mapstr.M{
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
	labels := mapstr.M{
		"runtime":  model.LabelValue("linux"),
		"quantile": model.LabelValue("0.25"),
	}
	labels2 := mapstr.M{
		"runtime":  model.LabelValue("linux"),
		"quantile": model.LabelValue("0.50"),
	}
	labels3 := mapstr.M{
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

	expected := mapstr.M{
		"go_gc_duration_seconds": mapstr.M{
			"value": float64(42),
		},
		"go_gc_duration_seconds_2": mapstr.M{
			"value": float64(46),
		},
		"labels": labels,
	}
	expected2 := mapstr.M{
		"go_gc_duration_seconds": mapstr.M{
			"value": float64(43),
		},
		"labels": labels2,
	}
	expected3 := mapstr.M{
		"go_gc_duration_seconds_count": mapstr.M{
			"counter": float64(45),
			"rate":    float64(0),
		},
		"go_gc_duration_seconds_sum": mapstr.M{
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

	expected = mapstr.M{
		"go_gc_duration_seconds": mapstr.M{
			"value": float64(52),
		},
		"go_gc_duration_seconds_2": mapstr.M{
			"value": float64(56),
		},
		"labels": labels,
	}
	expected2 = mapstr.M{
		"go_gc_duration_seconds": mapstr.M{
			"value": float64(53),
		},
		"labels": labels2,
	}
	expected3 = mapstr.M{
		"go_gc_duration_seconds_count": mapstr.M{
			"counter": float64(55),
			"rate":    float64(10),
		},
		"go_gc_duration_seconds_sum": mapstr.M{
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
	labels := mapstr.M{
		"runtime": model.LabelValue("linux"),
	}
	labels2 := mapstr.M{
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

	expected := mapstr.M{
		"http_request_duration_seconds": mapstr.M{
			"histogram": mapstr.M{
				"values": []float64{float64(0.125), float64(0.375), float64(0.5)},
				"counts": []uint64{uint64(0), uint64(0), uint64(0)},
			},
		},
		"http_request_duration_seconds_sum": mapstr.M{
			"counter": float64(45),
			"rate":    float64(0),
		},
		"http_request_duration_seconds_count": mapstr.M{
			"counter": float64(46),
			"rate":    float64(0),
		},
		"http_request_bytes": mapstr.M{
			"histogram": mapstr.M{
				"values": []float64{float64(0.125), float64(0.375), float64(0.5)},
				"counts": []uint64{uint64(0), uint64(0), uint64(0)},
			},
		},
		"http_request_bytes_sum": mapstr.M{
			"counter": float64(55),
			"rate":    float64(0),
		},
		"http_request_bytes_count": mapstr.M{
			"counter": float64(56),
			"rate":    float64(0),
		},
		"labels": labels,
	}
	expected2 := mapstr.M{
		"http_request_bytes": mapstr.M{
			"histogram": mapstr.M{
				"values": []float64{float64(0.125), float64(0.375), float64(0.5)},
				"counts": []uint64{uint64(0), uint64(0), uint64(0)},
			},
		},
		"http_request_bytes_sum": mapstr.M{
			"counter": float64(65),
			"rate":    float64(0),
		},
		"http_request_bytes_count": mapstr.M{
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

	expected = mapstr.M{
		"http_request_duration_seconds": mapstr.M{
			"histogram": mapstr.M{
				"values": []float64{float64(0.125), float64(0.375), float64(0.5)},
				"counts": []uint64{uint64(100), uint64(0), uint64(0)},
			},
		},
		"http_request_duration_seconds_sum": mapstr.M{
			"counter": float64(145),
			"rate":    float64(100),
		},
		"http_request_duration_seconds_count": mapstr.M{
			"counter": float64(146),
			"rate":    float64(100),
		},
		"http_request_bytes": mapstr.M{
			"histogram": mapstr.M{
				"values": []float64{float64(0.125), float64(0.375), float64(0.5)},
				"counts": []uint64{uint64(200), uint64(0), uint64(0)},
			},
		},
		"http_request_bytes_sum": mapstr.M{
			"counter": float64(255),
			"rate":    float64(200),
		},
		"http_request_bytes_count": mapstr.M{
			"counter": float64(256),
			"rate":    float64(200),
		},
		"labels": labels,
	}
	expected2 = mapstr.M{
		"http_request_bytes": mapstr.M{
			"histogram": mapstr.M{
				"values": []float64{float64(0.125), float64(0.375), float64(0.5)},
				"counts": []uint64{uint64(300), uint64(0), uint64(0)},
			},
		},
		"http_request_bytes_sum": mapstr.M{
			"counter": float64(365),
			"rate":    float64(300),
		},
		"http_request_bytes_count": mapstr.M{
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
	labels := mapstr.M{
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

	expected := mapstr.M{
		"net_conntrack_listener_conn_closed_mycounter": mapstr.M{
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

	expected = mapstr.M{
		"net_conntrack_listener_conn_closed_mycounter": mapstr.M{
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
	labels := mapstr.M{
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

	expected := mapstr.M{
		"net_conntrack_listener_conn_closed_myhistogram": mapstr.M{
			"histogram": mapstr.M{
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

	expected = mapstr.M{
		"net_conntrack_listener_conn_closed_myhistogram": mapstr.M{
			"histogram": mapstr.M{
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

func TestMetricsCount(t *testing.T) {
	tests := []struct {
		name     string
		samples  model.Samples
		expected map[string]int
	}{
		{
			name: "HTTP requests counter with multiple dimensions",
			samples: model.Samples{
				&model.Sample{
					Metric: model.Metric{"__name__": "http_requests_total", "method": "GET", "status": "200", "path": "/api/v1/users"},
					Value:  100,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "http_requests_total", "method": "POST", "status": "201", "path": "/api/v1/users"},
					Value:  50,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "http_requests_total", "method": "GET", "status": "404", "path": "/api/v1/products"},
					Value:  10,
				},
			},
			expected: map[string]int{
				`{"method":"GET","path":"/api/v1/users","status":"200"}`:    1,
				`{"method":"POST","path":"/api/v1/users","status":"201"}`:   1,
				`{"method":"GET","path":"/api/v1/products","status":"404"}`: 1,
			},
		},
		{
			name: "CPU and memory usage gauges",
			samples: model.Samples{
				&model.Sample{
					Metric: model.Metric{"__name__": "node_cpu_usage_percent", "cpu": "0", "mode": "user"},
					Value:  25.5,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "node_cpu_usage_percent", "cpu": "0", "mode": "system"},
					Value:  10.2,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "node_memory_usage_bytes", "type": "used"},
					Value:  4294967296, // 4GB
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "node_memory_usage_bytes", "type": "free"},
					Value:  8589934592, // 8GB
				},
			},
			expected: map[string]int{
				`{"cpu":"0","mode":"user"}`:   1,
				`{"cpu":"0","mode":"system"}`: 1,
				`{"type":"used"}`:             1,
				`{"type":"free"}`:             1,
			},
		},
		{
			name: "Request duration histogram",
			samples: model.Samples{
				&model.Sample{
					Metric: model.Metric{"__name__": "http_request_duration_seconds_bucket", "le": "0.1", "handler": "/home"},
					Value:  200,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "http_request_duration_seconds_bucket", "le": "0.5", "handler": "/home"},
					Value:  400,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "http_request_duration_seconds_bucket", "le": "+Inf", "handler": "/home"},
					Value:  500,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "http_request_duration_seconds_sum", "handler": "/home"},
					Value:  120.5,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "http_request_duration_seconds_count", "handler": "/home"},
					Value:  500,
				},
			},
			expected: map[string]int{
				`{"handler":"/home"}`: 3,
			},
		},
		{
			name: "Mix of counter, gauge, and histogram",
			samples: model.Samples{
				&model.Sample{
					Metric: model.Metric{"__name__": "http_requests_total", "method": "GET", "status": "200"},
					Value:  100,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "cpu_usage", "core": "0"},
					Value:  45.5,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "request_duration_seconds_bucket", "le": "0.1"},
					Value:  30,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "request_duration_seconds_bucket", "le": "0.5"},
					Value:  50,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "request_duration_seconds_sum"},
					Value:  75.5,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "request_duration_seconds_count"},
					Value:  60,
				},
			},
			expected: map[string]int{
				`{"method":"GET","status":"200"}`: 1,
				`{"core":"0"}`:                    1,
				`{}`:                              3,
			},
		},
		{
			name: "Duplicate labels and distinct labels",
			samples: model.Samples{
				&model.Sample{
					Metric: model.Metric{"__name__": "api_calls", "endpoint": "/users", "method": "GET"},
					Value:  50,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "api_calls", "endpoint": "/users", "method": "POST"},
					Value:  30,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "api_calls", "endpoint": "/products", "method": "GET"},
					Value:  40,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "system_load", "host": "server1"},
					Value:  1.5,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "system_load", "host": "server2"},
					Value:  2.0,
				},
			},
			expected: map[string]int{
				`{"endpoint":"/users","method":"GET"}`:    1,
				`{"endpoint":"/users","method":"POST"}`:   1,
				`{"endpoint":"/products","method":"GET"}`: 1,
				`{"host":"server1"}`:                      1,
				`{"host":"server2"}`:                      1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := remoteWriteTypedGenerator{
				metricsCount: true,
				counterCache: xcollector.NewCounterCache(time.Minute),
			}

			events := generator.GenerateEvents(tt.samples)

			for _, event := range events {
				count, ok := event.RootFields["metrics_count"]
				assert.True(t, ok, "metrics_count should be present")

				labels, ok := event.ModuleFields["labels"].(mapstr.M)
				if !ok {
					labels = mapstr.M{} // If no labels, create an empty map so that we can handle metrics with no labels
				}

				labelsHash := labels.String()

				expectedCount, ok := tt.expected[labelsHash]
				assert.True(t, ok, "should have an expected count for these labels")
				assert.Equal(t, expectedCount, count, "metrics_count should match expected value for labels %v", labels)
			}
		})
	}
}
