// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package remote_write

import (
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/common/model"

	"github.com/elastic/beats/v7/metricbeat/mb"
)

var benchmarkEventCount int

type benchmarkCapacityChecker interface {
	CheckCapacity(model.Samples) error
}

type benchmarkFlushCapable interface {
	FlushExpired(time.Time) map[string]mb.Event
}

// BenchmarkHistogramAssemblyPipeline compares request-local histogram conversion
// with cross-request histogram assembly on the same revision.
func BenchmarkHistogramAssemblyPipeline(b *testing.B) {
	modes := []struct {
		name            string
		assemblyEnabled bool
	}{
		{name: "assembly_disabled", assemblyEnabled: false},
		{name: "assembly_enabled", assemblyEnabled: true},
	}
	workloads := []struct {
		name  string
		build func() []model.Samples
	}{
		{
			name: "no_histograms_1000_samples",
			build: func() []model.Samples {
				return []model.Samples{benchmarkNonHistogramSamples(1_000)}
			},
		},
		{
			name: "complete_100_histograms_10_buckets",
			build: func() []model.Samples {
				return benchmarkHistogramSamples(100, 10, false)
			},
		},
		{
			name: "split_100_histograms_10_buckets",
			build: func() []model.Samples {
				return benchmarkHistogramSamples(100, 10, true)
			},
		},
		{
			name: "high_cardinality_1000_histograms_10_buckets",
			build: func() []model.Samples {
				return benchmarkHistogramSamples(1_000, 10, true)
			},
		},
	}

	for _, mode := range modes {
		b.Run(mode.name, func(b *testing.B) {
			for _, workload := range workloads {
				b.Run(workload.name, func(b *testing.B) {
					b.ReportAllocs()
					for range b.N {
						b.StopTimer()
						generator := benchmarkTypedGeneratorWithAssembly(false, mode.assemblyEnabled)
						batches := workload.build()
						b.StartTimer()

						eventCount := 0
						for _, batch := range batches {
							if checker, ok := any(generator).(benchmarkCapacityChecker); ok {
								if err := checker.CheckCapacity(batch); err != nil {
									b.Fatalf("capacity check failed: %v", err)
								}
							}
							eventCount += len(generator.GenerateEvents(batch))
						}
						if flusher, ok := any(generator).(benchmarkFlushCapable); ok {
							eventCount += len(flusher.FlushExpired(time.Now().Add(time.Minute)))
						}

						b.StopTimer()
						generator.counterCache.Stop()
						benchmarkEventCount = eventCount
					}
				})
			}
		})
	}
}

func benchmarkNonHistogramSamples(count int) model.Samples {
	samples := make(model.Samples, 0, count)
	timestamp := model.Time(1_000_000)
	for i := 0; i < count; i++ {
		samples = append(samples, &model.Sample{
			Metric: model.Metric{
				"__name__": "benchmark_gauge",
				"series":   model.LabelValue(strconv.Itoa(i)),
			},
			Value:     model.SampleValue(i),
			Timestamp: timestamp,
		})
	}
	return samples
}

func benchmarkHistogramSamples(histograms, finiteBuckets int, split bool) []model.Samples {
	batchCount := 1
	if split {
		batchCount = 2
	}
	batches := make([]model.Samples, batchCount)
	timestamp := model.Time(1_000_000)

	for histogram := 0; histogram < histograms; histogram++ {
		series := model.LabelValue(strconv.Itoa(histogram))
		for bucket := 0; bucket < finiteBuckets; bucket++ {
			batch := 0
			if split {
				batch = bucket % batchCount
			}
			batches[batch] = append(batches[batch], &model.Sample{
				Metric: model.Metric{
					"__name__": "benchmark_duration_seconds_bucket",
					"series":   series,
					"le":       model.LabelValue(strconv.Itoa(bucket + 1)),
				},
				Value:     model.SampleValue(bucket + 1),
				Timestamp: timestamp,
			})
		}

		infBatch := batchCount - 1
		batches[infBatch] = append(batches[infBatch], &model.Sample{
			Metric: model.Metric{
				"__name__": "benchmark_duration_seconds_bucket",
				"series":   series,
				"le":       "+Inf",
			},
			Value:     model.SampleValue(finiteBuckets + 1),
			Timestamp: timestamp,
		})
	}
	return batches
}
