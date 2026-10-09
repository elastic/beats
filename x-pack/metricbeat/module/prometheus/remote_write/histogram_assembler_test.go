// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote_write

import (
	"errors"
	"maps"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	rw "github.com/elastic/beats/v7/metricbeat/module/prometheus/remote_write"
	xcollector "github.com/elastic/beats/v7/x-pack/metricbeat/module/prometheus/collector"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func testClock(start time.Time) (func() time.Time, func(time.Time)) {
	current := start
	return func() time.Time { return current }, func(t time.Time) { current = t }
}

func newTestTypedGenerator(t *testing.T, cfg histogramAssemblyConfig, start time.Time) (*remoteWriteTypedGenerator, func(time.Time)) {
	t.Helper()
	nowFn, setNow := testClock(start)
	counters := xcollector.NewCounterCache(time.Minute)
	g := &remoteWriteTypedGenerator{
		counterCache:    counters,
		rateCounters:    true,
		assemblyConfig:  cfg,
		now:             nowFn,
		retainedFlushes: make(map[string]mb.Event),
	}
	g.assembler = newHistogramAssembler(cfg, nil)
	g.counterCache.Start()
	t.Cleanup(g.counterCache.Stop)
	return g, setNow
}

func mergeEvents(base, flush map[string]mb.Event) map[string]mb.Event {
	out := make(map[string]mb.Event, len(base)+len(flush))
	maps.Copy(out, base)
	for k, v := range flush {
		if existing, ok := out[k]; ok {
			existing.ModuleFields.Update(v.ModuleFields)
			if len(v.RootFields) > 0 {
				if existing.RootFields == nil {
					existing.RootFields = mapstr.M{}
				}
				existing.RootFields.Update(v.RootFields)
			}
			out[k] = existing
		} else {
			out[k] = v
		}
	}
	for k, e := range out {
		if _, ok := e.RootFields["metrics_count"]; !ok {
			continue
		}
		if _, hasLabels := e.ModuleFields["labels"]; hasLabels {
			e.RootFields["metrics_count"] = len(e.ModuleFields) - 1
		} else {
			e.RootFields["metrics_count"] = len(e.ModuleFields)
		}
		out[k] = e
	}
	return out
}

func promBucketSample(name string, labelVals map[string]string, value float64, ts model.Time) *model.Sample {
	metric := map[model.LabelName]model.LabelValue{"__name__": model.LabelValue(name)}
	for k, v := range labelVals {
		metric[model.LabelName(k)] = model.LabelValue(v)
	}
	return &model.Sample{Metric: metric, Value: model.SampleValue(value), Timestamp: ts}
}

func TestHistogramAssemblerSameRequestBuffering(t *testing.T) {
	cfg := defaultHistogramAssemblyConfig()
	start := time.Unix(100, 0)
	g, setNow := newTestTypedGenerator(t, cfg, start)
	ts := model.TimeFromUnix(100)

	metrics := model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 10, ts),
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.50"}, 20, ts),
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "+Inf"}, 30, ts),
	}
	setNow(ts.Time())
	events := g.GenerateEvents(metrics)
	assert.Empty(t, events, "bucket-only batch must not emit events immediately")

	setNow(ts.Time().Add(cfg.QuietPeriod + time.Millisecond))
	flushed := g.FlushExpired(g.now())
	require.Len(t, flushed, 1, "quiet flush after +Inf should emit one histogram event")
	for _, ev := range flushed {
		hist := ev.ModuleFields["http_request_duration_seconds"].(mapstr.M)["histogram"].(mapstr.M)
		assert.Equal(t, []float64{0.125, 0.375, 0.5}, hist["values"], "same-request buckets must assemble all centroids")
	}
}

func TestHistogramAssemblerCrossCallMerge(t *testing.T) {
	cfg := defaultHistogramAssemblyConfig()
	start := time.Unix(200, 0)
	g, setNow := newTestTypedGenerator(t, cfg, start)
	ts := model.TimeFromUnix(200)

	setNow(ts.Time())
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 10, ts),
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.50"}, 20, ts),
	})
	setNow(ts.Time().Add(2 * time.Second))
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "+Inf"}, 30, ts),
	})

	flushed := g.FlushExpired(ts.Time().Add(2*time.Second + cfg.QuietPeriod + time.Millisecond))
	require.Len(t, flushed, 1)
	labels := mapstr.M{"runtime": model.LabelValue("linux")}
	hist := flushed[labels.String()+ts.Time().String()].ModuleFields["http_request_duration_seconds"].(mapstr.M)["histogram"].(mapstr.M)
	assert.Equal(t, []float64{0.125, 0.375, 0.5}, hist["values"])
}

func TestHistogramAssemblerHardTimeoutPartial(t *testing.T) {
	cfg := defaultHistogramAssemblyConfig()
	start := time.Unix(300, 0)
	g, setNow := newTestTypedGenerator(t, cfg, start)
	ts := model.TimeFromUnix(300)

	setNow(ts.Time())
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 10, ts),
	})

	flushed := g.FlushExpired(ts.Time().Add(cfg.HardTimeout))
	require.Len(t, flushed, 1, "hard timeout must flush partial histogram")
	assert.Equal(t, uint64(1), g.assembler.stats.FlushesPartial)
}

func TestHistogramAssemblerDuplicateLeGreatestCumulative(t *testing.T) {
	cfg := defaultHistogramAssemblyConfig()
	a := newHistogramAssembler(cfg, nil)
	now := time.Unix(0, 0)
	labels := mapstr.M{"runtime": model.LabelValue("linux")}
	ts := now
	a.ingest(now, bucketSample{
		bucketMetricName: "http_request_duration_seconds_bucket",
		labels:           labels,
		timestamp:        ts,
		buckets:          []bucketUpdate{{upperBound: 0.5, cumulativeCount: 20}},
	})
	a.ingest(now, bucketSample{
		bucketMetricName: "http_request_duration_seconds_bucket",
		labels:           labels,
		timestamp:        ts,
		buckets:          []bucketUpdate{{upperBound: 0.5, cumulativeCount: 15}},
	})
	a.ingest(now, bucketSample{
		bucketMetricName: "http_request_duration_seconds_bucket",
		labels:           labels,
		timestamp:        ts,
		buckets:          []bucketUpdate{{upperBound: 0.5, cumulativeCount: 25}},
	})
	key := histogramIdentityFromParts("http_request_duration_seconds_bucket", labels, ts).key()
	assert.Equal(t, 25.0, a.pending[key].buckets[0.5].GetCumulativeCount())
}

func TestHistogramAssemblerOrdersBucketsBeforeConversion(t *testing.T) {
	buckets := map[float64]*p.Bucket{
		math.Inf(1): {UpperBound: new(math.Inf(1)), CumulativeCount: new(30.0)},
		0.50:        {UpperBound: new(0.50), CumulativeCount: new(20.0)},
		-1:          {UpperBound: new(-1.0), CumulativeCount: new(5.0)},
		0.25:        {UpperBound: new(0.25), CumulativeCount: new(10.0)},
	}

	ordered := histogramBucketsInOrder(buckets)

	require.Len(t, ordered, 4)
	assert.Equal(t, []float64{-1, 0.25, 0.50, math.Inf(1)}, []float64{
		ordered[0].GetUpperBound(),
		ordered[1].GetUpperBound(),
		ordered[2].GetUpperBound(),
		ordered[3].GetUpperBound(),
	})
}

func TestCheckCapacityRejectsWithoutMutation(t *testing.T) {
	cfg := histogramAssemblyConfig{
		QuietPeriod:          time.Second,
		HardTimeout:          10 * time.Second,
		MaxPendingHistograms: 1,
		MaxPendingBuckets:    10,
	}
	start := time.Unix(400, 0)
	g, setNow := newTestTypedGenerator(t, cfg, start)
	ts := model.TimeFromUnix(400)

	setNow(ts.Time())
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 1, ts),
	})

	second := model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "darwin", "le": "0.25"}, 1, ts),
	}
	err := g.CheckCapacity(second)
	require.ErrorIs(t, err, rw.ErrRemoteWriteCapacityExceeded)
	snap := g.assembler.statsSnapshot()
	assert.Equal(t, 1, snap.PendingHistograms, "pending histogram count must be unchanged after rejected batch")
	assert.Equal(t, 1, snap.PendingBuckets, "pending bucket count must be unchanged after rejected batch")
}

func TestCheckCapacityRejectsBucketOverflowWithoutMutation(t *testing.T) {
	cfg := histogramAssemblyConfig{
		QuietPeriod:          time.Second,
		HardTimeout:          10 * time.Second,
		MaxPendingHistograms: 10,
		MaxPendingBuckets:    2,
	}
	start := time.Unix(401, 0)
	g, setNow := newTestTypedGenerator(t, cfg, start)
	ts := model.TimeFromUnix(401)

	setNow(ts.Time())
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 1, ts),
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.50"}, 2, ts),
	})
	before := g.assembler.statsSnapshot()

	overflow := model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "+Inf"}, 3, ts),
	}
	require.ErrorIs(t, g.CheckCapacity(overflow), rw.ErrRemoteWriteCapacityExceeded)
	after := g.assembler.statsSnapshot()
	assert.Equal(t, before.PendingHistograms, after.PendingHistograms, "histogram capacity rejection must not add pending histograms")
	assert.Equal(t, before.PendingBuckets, after.PendingBuckets, "bucket capacity rejection must not add pending buckets")
	assert.Equal(t, before.PendingBuckets, 2, "preflight must observe existing pending buckets")
}

func TestHistogramAssemblerTombstoneLateDrop(t *testing.T) {
	cfg := defaultHistogramAssemblyConfig()
	start := time.Unix(500, 0)
	g, setNow := newTestTypedGenerator(t, cfg, start)
	ts := model.TimeFromUnix(500)

	setNow(ts.Time())
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 10, ts),
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "+Inf"}, 30, ts),
	})
	setNow(ts.Time().Add(cfg.QuietPeriod + time.Millisecond))
	_ = g.FlushExpired(g.now())

	setNow(g.now().Add(time.Millisecond))
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.50"}, 20, ts),
	})
	assert.Equal(t, uint64(1), g.assembler.stats.LateDrops)
	flushed := g.FlushExpired(g.now())
	assert.Empty(t, flushed, "late bucket during tombstone must not reopen histogram")
}

func TestHistogramAssemblerRetainUnpublishedRetry(t *testing.T) {
	cfg := defaultHistogramAssemblyConfig()
	g, _ := newTestTypedGenerator(t, cfg, time.Unix(0, 0))
	key := "retry-key"
	ev := map[string]mb.Event{key: {RootFields: mapstr.M{"kept": true}}}
	g.RetainUnpublishedFlushEvents(ev)
	first := g.FlushExpired(time.Unix(1, 0))
	require.Contains(t, first, key)
	g.RetainUnpublishedFlushEvents(map[string]mb.Event{key: first[key]})
	second := g.FlushExpired(time.Unix(2, 0))
	require.Contains(t, second, key, "retained flush events must be retried on next flush")
}

func TestHistogramAssemblerShutdownDrop(t *testing.T) {
	cfg := defaultHistogramAssemblyConfig()
	g, setNow := newTestTypedGenerator(t, cfg, time.Unix(600, 0))
	ts := model.TimeFromUnix(600)
	setNow(ts.Time())
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 1, ts),
	})
	g.RetainUnpublishedFlushEvents(map[string]mb.Event{"k": {}})
	g.Stop()
	stats := g.HistogramAssemblyStats()
	assert.Equal(t, uint64(1), stats.ShutdownDroppedHists)
	assert.Equal(t, uint64(1), stats.ShutdownDroppedEvents)
}

func TestHistogramAssemblerBoundedTombstones(t *testing.T) {
	cfg := histogramAssemblyConfig{
		QuietPeriod:          time.Millisecond,
		HardTimeout:          time.Hour,
		MaxPendingHistograms: 2,
		MaxPendingBuckets:    100,
	}
	a := newHistogramAssembler(cfg, nil)
	now := time.Unix(0, 0)
	for i := range 5 {
		labels := mapstr.M{"i": model.LabelValue(strconv.Itoa(i))}
		ts := now.Add(time.Duration(i) * time.Second)
		a.ingest(now, bucketSample{
			bucketMetricName: "m_bucket",
			labels:           labels,
			timestamp:        ts,
			buckets:          []bucketUpdate{{upperBound: math.Inf(1), cumulativeCount: 1}},
		})
		_ = a.flushExpired(now.Add(time.Second), xcollector.NewCounterCache(time.Minute), false)
	}
	assert.LessOrEqual(t, len(a.tombstones), 2, "tombstones must stay bounded by max pending histograms")
}

func TestGenerateEventsCounterUnaffectedByAssembly(t *testing.T) {
	cfg := defaultHistogramAssemblyConfig()
	g, setNow := newTestTypedGenerator(t, cfg, time.Unix(700, 0))
	ts := model.Time(424242)
	labels := mapstr.M{"listener_name": model.LabelValue("http")}
	setNow(ts.Time())
	metrics := model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(42),
			Timestamp: ts,
		},
	}
	events := g.GenerateEvents(metrics)
	require.Len(t, events, 1)
	assert.Equal(t, float64(42), events[labels.String()+ts.Time().String()].ModuleFields["net_conntrack_listener_conn_closed_total"].(mapstr.M)["counter"])
}

func TestNextFlushIntervalDeterministic(t *testing.T) {
	g, _ := newTestTypedGenerator(t, defaultHistogramAssemblyConfig(), time.Unix(0, 0))
	assert.Equal(t, 2500*time.Millisecond, g.NextFlushInterval(), "NextFlushInterval should be min(quiet,hard)/2 with floor")
}

func TestCheckCapacityIncludesRetainedFlushEvents(t *testing.T) {
	cfg := histogramAssemblyConfig{
		QuietPeriod:          time.Second,
		HardTimeout:          10 * time.Second,
		MaxPendingHistograms: 1,
		MaxPendingBuckets:    10,
	}
	g, setNow := newTestTypedGenerator(t, cfg, time.Unix(800, 0))
	ts := model.TimeFromUnix(800)

	setNow(ts.Time())
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 1, ts),
	})
	flushed := g.FlushExpired(ts.Time().Add(cfg.HardTimeout))
	require.NotEmpty(t, flushed)
	g.RetainUnpublishedFlushEvents(flushed)

	blocking := model.Samples{
		promBucketSample("other_metric_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 1, ts),
	}
	require.ErrorIs(t, g.CheckCapacity(blocking), rw.ErrRemoteWriteCapacityExceeded)

	g.RetainUnpublishedFlushEvents(nil)
	require.NoError(t, g.CheckCapacity(blocking), "capacity must recover after retained flush events are cleared")
}

func TestCheckCapacityIncludesRetainedBucketCounts(t *testing.T) {
	cfg := histogramAssemblyConfig{
		QuietPeriod:          time.Second,
		HardTimeout:          10 * time.Second,
		MaxPendingHistograms: 10,
		MaxPendingBuckets:    3,
	}
	g, setNow := newTestTypedGenerator(t, cfg, time.Unix(810, 0))
	ts := model.TimeFromUnix(810)

	setNow(ts.Time())
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 1, ts),
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.50"}, 2, ts),
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "+Inf"}, 3, ts),
	})
	flushed := g.FlushExpired(ts.Time().Add(cfg.QuietPeriod + time.Millisecond))
	g.RetainUnpublishedFlushEvents(flushed)

	extra := model.Samples{
		promBucketSample("other_metric_bucket", map[string]string{"zone": "a", "le": "1"}, 1, ts),
	}
	require.ErrorIs(t, g.CheckCapacity(extra), rw.ErrRemoteWriteCapacityExceeded)
}

func TestRetainUnpublishedFlushEventsDeterministicOverflow(t *testing.T) {
	cfg := histogramAssemblyConfig{
		QuietPeriod:          time.Second,
		HardTimeout:          time.Second,
		MaxPendingHistograms: 1,
		MaxPendingBuckets:    100,
	}
	g, _ := newTestTypedGenerator(t, cfg, time.Unix(0, 0))

	events := map[string]mb.Event{
		"b-key": {ModuleFields: mapstr.M{
			"metric_b_bucket": mapstr.M{"histogram": mapstr.M{"values": []float64{1}, "counts": []uint64{1}}},
		}},
		"a-key": {ModuleFields: mapstr.M{
			"metric_a_bucket": mapstr.M{"histogram": mapstr.M{"values": []float64{1}, "counts": []uint64{1}}},
		}},
	}
	g.RetainUnpublishedFlushEvents(events)
	require.Len(t, g.retainedFlushes, 1)
	assert.Contains(t, g.retainedFlushes, "a-key", "retain overflow must keep lowest sorted key")
	stats := g.HistogramAssemblyStats()
	assert.Equal(t, uint64(1), stats.RetentionDrops)
	assert.Equal(t, uint64(1), stats.RetentionBucketDrops)
}

func TestHistogramAssemblerTombstoneExpiresBeforeReaccept(t *testing.T) {
	cfg := histogramAssemblyConfig{
		QuietPeriod:          time.Second,
		HardTimeout:          10 * time.Second,
		MaxPendingHistograms: 10,
		MaxPendingBuckets:    100,
	}
	g, setNow := newTestTypedGenerator(t, cfg, time.Unix(900, 0))
	ts := model.TimeFromUnix(900)
	setNow(ts.Time())

	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "+Inf"}, 1, ts),
	})
	flushAt := ts.Time().Add(cfg.QuietPeriod + time.Millisecond)
	setNow(flushAt)
	g.FlushExpired(flushAt)
	require.Equal(t, 1, g.HistogramAssemblyStats().Tombstones, "flush must leave a tombstone for the identity")

	setNow(flushAt.Add(cfg.HardTimeout - time.Millisecond))
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.5"}, 2, ts),
	})
	assert.Equal(t, uint64(1), g.assembler.stats.LateDrops, "same identity inside hard timeout after flush must be dropped")

	setNow(flushAt.Add(cfg.HardTimeout + time.Millisecond))
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.5"}, 2, ts),
	})
	assert.Equal(t, 1, g.assembler.statsSnapshot().PendingHistograms, "exact identity must be accepted strictly after hard timeout")
}

func TestStopDoesNotPublishBufferedHistograms(t *testing.T) {
	cfg := defaultHistogramAssemblyConfig()
	g, setNow := newTestTypedGenerator(t, cfg, time.Unix(950, 0))
	ts := model.TimeFromUnix(950)
	setNow(ts.Time())
	g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 1, ts),
	})
	g.Stop()
	assert.Equal(t, 0, g.assembler.statsSnapshot().PendingHistograms)
	stats := g.HistogramAssemblyStats()
	assert.Equal(t, uint64(1), stats.ShutdownDroppedHists)
}

func TestGenerateEventsImmediateSumSeparateFromBufferedHistogram(t *testing.T) {
	cfg := defaultHistogramAssemblyConfig()
	g, setNow := newTestTypedGenerator(t, cfg, time.Unix(960, 0))
	ts := model.TimeFromUnix(960)
	setNow(ts.Time())

	immediate := g.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 1, ts),
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "http_request_duration_seconds_sum",
				"runtime":  "linux",
			},
			Value:     2,
			Timestamp: ts,
		},
	})
	require.Len(t, immediate, 1, "sum must emit immediately even while buckets buffer")
	for _, ev := range immediate {
		assert.Contains(t, ev.ModuleFields, "http_request_duration_seconds_sum")
		assert.NotContains(t, ev.ModuleFields, "http_request_duration_seconds")
	}
}

func TestProcessOwnerLoopBatchGroupsBucketsByIdentity(t *testing.T) {
	cfg := defaultHistogramAssemblyConfig()
	start := time.Unix(970, 0)
	g, setNow := newTestTypedGenerator(t, cfg, start)
	ts := model.TimeFromUnix(970)
	setNow(ts.Time())

	events, err := g.ProcessOwnerLoopBatch(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "zone": "a", "le": "0.25"}, 10, ts),
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "zone": "b", "le": "0.25"}, 11, ts),
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "zone": "a", "le": "0.50"}, 20, ts),
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "zone": "b", "le": "+Inf"}, 30, ts),
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "zone": "a", "le": "+Inf"}, 30, ts),
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__": "cpu_usage",
				"runtime":  "linux",
			},
			Value:     7,
			Timestamp: ts,
		},
	})
	require.NoError(t, err)
	require.Len(t, events, 1, "only immediate non-histogram metrics must emit now")
	assert.Equal(t, 2, g.assembler.statsSnapshot().PendingHistograms, "two identities must be buffered")

	setNow(ts.Time().Add(cfg.QuietPeriod + time.Millisecond))
	flushed := g.FlushExpired(g.now())
	require.Len(t, flushed, 2, "quiet flush must emit one event per identity")
}

func TestProcessOwnerLoopBatchDuplicateBoundsKeepGreatest(t *testing.T) {
	cfg := defaultHistogramAssemblyConfig()
	start := time.Unix(971, 0)
	g, setNow := newTestTypedGenerator(t, cfg, start)
	ts := model.TimeFromUnix(971)
	setNow(ts.Time())

	_, err := g.ProcessOwnerLoopBatch(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.5"}, 10, ts),
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.5"}, 25, ts),
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "+Inf"}, 30, ts),
	})
	require.NoError(t, err)
	snap := g.assembler.statsSnapshot()
	assert.Equal(t, 1, snap.PendingHistograms)
	assert.Equal(t, 2, snap.PendingBuckets, "duplicate le in one request must count once")

	var entry *pendingHistogram
	for _, p := range g.assembler.pending {
		entry = p
		break
	}
	require.NotNil(t, entry)
	require.Contains(t, entry.buckets, 0.5)
	assert.Equal(t, float64(25), entry.buckets[0.5].GetCumulativeCount())
}

func TestProcessOwnerLoopBatchCapacityRejectionNoMutation(t *testing.T) {
	cfg := histogramAssemblyConfig{
		QuietPeriod:          time.Second,
		HardTimeout:          10 * time.Second,
		MaxPendingHistograms: 1,
		MaxPendingBuckets:    10,
	}
	start := time.Unix(972, 0)
	g, setNow := newTestTypedGenerator(t, cfg, start)
	ts := model.TimeFromUnix(972)
	setNow(ts.Time())

	_, err := g.ProcessOwnerLoopBatch(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 1, ts),
	})
	require.NoError(t, err)
	before := g.assembler.statsSnapshot()

	rejectedCounter := &model.Sample{
		Metric: map[model.LabelName]model.LabelValue{
			"__name__": "requests_total",
			"runtime":  "linux",
		},
		Value:     100,
		Timestamp: ts,
	}
	_, err = g.ProcessOwnerLoopBatch(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "darwin", "le": "0.25"}, 1, ts),
		rejectedCounter,
	})
	require.ErrorIs(t, err, rw.ErrRemoteWriteCapacityExceeded)
	after := g.assembler.statsSnapshot()
	assert.Equal(t, before.PendingHistograms, after.PendingHistograms)
	assert.Equal(t, before.PendingBuckets, after.PendingBuckets)

	// If the rejected batch had seeded the cache at 100, this observation would report rate 50.
	// With deferred counter-cache updates, the first successful observation still has rate 0.
	acceptedCounter := &model.Sample{
		Metric: map[model.LabelName]model.LabelValue{
			"__name__": "requests_total",
			"runtime":  "linux",
		},
		Value:     150,
		Timestamp: ts,
	}
	events, err := g.ProcessOwnerLoopBatch(model.Samples{acceptedCounter})
	require.NoError(t, err)
	require.Len(t, events, 1)
	labels := mapstr.M{"runtime": model.LabelValue("linux")}
	metric := events[labels.String()+ts.Time().String()].ModuleFields["requests_total"].(mapstr.M)
	assert.Equal(t, float64(150), metric["counter"])
	assert.Equal(t, float64(0), metric["rate"], "rejected batch must not seed counter cache")
}

// TestProcessOwnerLoopBatchUsesSingleNowAcrossCapacityAndCommit locks the
// capacity-check and commit path to a single now() snapshot. If those steps
// each call now() independently, a tombstone can expire between them: admission
// treats the identity as free (still tombstoned), then ingest reopens it and
// pending can exceed max_pending_histograms. NeverExceedsMaxPending alone does
// not cover this boundary.
func TestProcessOwnerLoopBatchUsesSingleNowAcrossCapacityAndCommit(t *testing.T) {
	cfg := histogramAssemblyConfig{
		QuietPeriod:          time.Second,
		HardTimeout:          2 * time.Second,
		MaxPendingHistograms: 2,
		MaxPendingBuckets:    100,
	}
	start := time.Unix(5000, 0)
	g, setNow := newTestTypedGenerator(t, cfg, start)
	ts := model.TimeFromUnix(5000)

	setNow(start)
	_, err := g.ProcessOwnerLoopBatch(model.Samples{
		promBucketSample("m_bucket", map[string]string{"id": "a", "le": "+Inf"}, 1, ts),
		promBucketSample("m_bucket", map[string]string{"id": "b", "le": "+Inf"}, 1, ts),
	})
	require.NoError(t, err)

	flushAt := start.Add(cfg.QuietPeriod + time.Millisecond)
	setNow(flushAt)
	require.Len(t, g.FlushExpired(g.now()), 2)

	ts2 := model.TimeFromUnix(5001)
	_, err = g.ProcessOwnerLoopBatch(model.Samples{
		promBucketSample("m_bucket", map[string]string{"id": "c", "le": "+Inf"}, 1, ts2),
		promBucketSample("m_bucket", map[string]string{"id": "d", "le": "+Inf"}, 1, ts2),
	})
	require.NoError(t, err)
	require.Equal(t, 2, g.assembler.statsSnapshot().PendingHistograms)

	// If capacity and commit each called now() separately, the second call would
	// land after tombstone expiry and reopen identities without admission cost.
	beforeExpiry := flushAt.Add(cfg.HardTimeout - time.Millisecond)
	afterExpiry := flushAt.Add(cfg.HardTimeout + time.Millisecond)
	var nowCalls int
	g.now = func() time.Time {
		nowCalls++
		if nowCalls == 1 {
			return beforeExpiry
		}
		return afterExpiry
	}

	_, err = g.ProcessOwnerLoopBatch(model.Samples{
		promBucketSample("m_bucket", map[string]string{"id": "a", "le": "+Inf"}, 1, ts),
		promBucketSample("m_bucket", map[string]string{"id": "b", "le": "+Inf"}, 1, ts),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, nowCalls, "capacity and commit must share one now() snapshot")
	assert.Equal(t, 2, g.assembler.statsSnapshot().PendingHistograms,
		"active tombstones at the snapshot time must late-drop instead of exceeding max pending")
	assert.Equal(t, uint64(2), g.assembler.stats.LateDrops)
}

func TestProcessOwnerLoopBatchNeverExceedsMaxPending(t *testing.T) {
	cfg := histogramAssemblyConfig{
		QuietPeriod:          time.Hour,
		HardTimeout:          time.Hour,
		MaxPendingHistograms: 50,
		MaxPendingBuckets:    10_000,
	}
	g, setNow := newTestTypedGenerator(t, cfg, time.Unix(1000, 0))
	ts := model.TimeFromUnix(1000)
	setNow(ts.Time())

	accepted := 0
	rejected := 0
	for i := range 200 {
		samples := model.Samples{
			promBucketSample("http_request_duration_seconds_bucket", map[string]string{
				"runtime": "linux",
				"id":      strconv.Itoa(i),
				"le":      "0.25",
			}, 1, ts),
			promBucketSample("http_request_duration_seconds_bucket", map[string]string{
				"runtime": "linux",
				"id":      strconv.Itoa(i),
				"le":      "+Inf",
			}, 2, ts),
		}
		_, err := g.ProcessOwnerLoopBatch(samples)
		if errors.Is(err, rw.ErrRemoteWriteCapacityExceeded) {
			rejected++
			continue
		}
		require.NoError(t, err)
		accepted++
	}
	snap := g.assembler.statsSnapshot()
	assert.LessOrEqual(t, snap.PendingHistograms, cfg.MaxPendingHistograms,
		"pending histograms must never exceed max (accepted=%d rejected=%d)", accepted, rejected)
	assert.Equal(t, cfg.MaxPendingHistograms, snap.PendingHistograms, "should fill exactly to max")
	assert.Greater(t, rejected, 0, "later batches must be capacity-rejected")
}
