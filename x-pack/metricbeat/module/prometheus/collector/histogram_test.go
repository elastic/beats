// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// skipping tests on windows 32 bit versions, not supported
//go:build !integration && !windows && !386

package collector

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestPromHistogramToES tests that calling PromHistogramToES multiple
// times with the same cache produces each time the expected results.
func TestPromHistogramToES(t *testing.T) {
	type sample struct {
		histogram p.Histogram
		expected  mapstr.M
	}

	cases := map[string]struct {
		samples []sample
	}{
		"one histogram": {
			samples: []sample{
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(10),
						SampleSum:   proto.Float64(10),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(10),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{0},
						"values": []float64{0.495},
					},
				},
			},
		},
		"two histogram": {
			samples: []sample{
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(10),
						SampleSum:   proto.Float64(10),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(10),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{0},
						"values": []float64{0.495},
					},
				},
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(12),
						SampleSum:   proto.Float64(10.123),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(12),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{2},
						"values": []float64{0.495},
					},
				},
			},
		},
		"new bucket on the go": {
			samples: []sample{
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(10),
						SampleSum:   proto.Float64(10),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(10),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{0},
						"values": []float64{0.495},
					},
				},
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(13),
						SampleSum:   proto.Float64(15.23),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(12),
							},
							// New bucket on the go
							{
								UpperBound:      proto.Float64(9.99),
								CumulativeCount: proto.Float64(13),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{2, 0},
						"values": []float64{0.495, 5.49},
					},
				},
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(15),
						SampleSum:   proto.Float64(16.33),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(13),
							},
							{
								UpperBound:      proto.Float64(9.99),
								CumulativeCount: proto.Float64(15),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{1, 1},
						"values": []float64{0.495, 5.49},
					},
				},
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(16),
						SampleSum:   proto.Float64(16.33),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(13),
							},
							{
								UpperBound:      proto.Float64(9.99),
								CumulativeCount: proto.Float64(16),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{0, 1},
						"values": []float64{0.495, 5.49},
					},
				},
			},
		},
		"new smaller bucket on the go": {
			samples: []sample{
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(10),
						SampleSum:   proto.Float64(10),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(10),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{0},
						"values": []float64{0.495},
					},
				},
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(13),
						SampleSum:   proto.Float64(15.23),
						Bucket: []*p.Bucket{
							// New bucket on the go
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Float64(1),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(13),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{0, 2},
						"values": []float64{0.045, 0.54},
					},
				},
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(15),
						SampleSum:   proto.Float64(16.33),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Float64(2),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(15),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{1, 1},
						"values": []float64{0.045, 0.54},
					},
				},
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(16),
						SampleSum:   proto.Float64(16.33),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Float64(3),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(16),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{1, 0},
						"values": []float64{0.045, 0.54},
					},
				},
			},
		},
		"new bucket between two other buckets on the go": {
			samples: []sample{
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(10),
						SampleSum:   proto.Float64(10),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Float64(0),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(10),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{0, 0},
						"values": []float64{0.045, 0.54},
					},
				},
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(13),
						SampleSum:   proto.Float64(15.23),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Float64(1),
							},
							// New bucket
							{
								UpperBound:      proto.Float64(0.49),
								CumulativeCount: proto.Float64(2),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(13),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{1, 0, 1},
						"values": []float64{0.045, 0.29000000000000004, 0.74},
					},
				},
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(16),
						SampleSum:   proto.Float64(16.33),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Float64(2),
							},
							{
								UpperBound:      proto.Float64(0.49),
								CumulativeCount: proto.Float64(4),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(16),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{1, 1, 1},
						"values": []float64{0.045, 0.29000000000000004, 0.74},
					},
				},
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(18),
						SampleSum:   proto.Float64(16.33),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Float64(3),
							},
							{
								UpperBound:      proto.Float64(0.49),
								CumulativeCount: proto.Float64(5),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(18),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{1, 0, 1},
						"values": []float64{0.045, 0.29000000000000004, 0.74},
					},
				},
			},
		},
		"wrong buckets": {
			samples: []sample{
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(10),
						SampleSum:   proto.Float64(10),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Float64(10),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(8),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{0, 0},
						"values": []float64{0.045, 0.54},
					},
				},
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(12),
						SampleSum:   proto.Float64(10.45),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Float64(12),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Float64(8),
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{2, 0},
						"values": []float64{0.045, 0.54},
					},
				},
			},
		},
		"histogram with negative buckets": {
			samples: []sample{
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(30),
						SampleSum:   proto.Float64(5),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(-100),
								CumulativeCount: proto.Float64(2),
							},
							{
								UpperBound:      proto.Float64(-99),
								CumulativeCount: proto.Float64(10),
							},
							{
								UpperBound:      proto.Float64(0),
								CumulativeCount: proto.Float64(30),
							},
						},
					},
					expected: mapstr.M{
						// rate of the first bucket is always 0, meaning that it was not increased as it is the first
						// count rate: [0, 8, 20]
						"counts": []uint64{0, 0, 0},
						"values": []float64{-100, -99.5, -49.5},
					},
				},
				{
					histogram: p.Histogram{
						SampleCount: proto.Float64(100),
						SampleSum:   proto.Float64(20),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(-100),
								CumulativeCount: proto.Float64(5),
							},
							{
								UpperBound:      proto.Float64(-99),
								CumulativeCount: proto.Float64(16),
							},
							{
								UpperBound:      proto.Float64(0),
								CumulativeCount: proto.Float64(100),
							},
						},
					},
					expected: mapstr.M{
						// counts calculation:
						// UpperBound -100: 5-2
						// UpperBound -99: 16 - 5 (undo accumulation) - 8 (calculate rate)
						// UpperBound 0: 100 - 16 (undo accumulation) - 20 (calculate rate)
						"counts": []uint64{3, 3, 64},
						"values": []float64{-100, -99.5, -49.5},
					},
				},
			},
		},
	}

	metricName := "somemetric"
	labels := mapstr.M{}

	for title, c := range cases {
		t.Run(title, func(t *testing.T) {
			cache := NewCounterCache(120 * time.Minute)

			for i, s := range c.samples {
				t.Logf("#%d: %+v", i, s.histogram)
				result := PromHistogramToES(cache, metricName, labels, &s.histogram)
				assert.EqualValues(t, s.expected, result)
			}
		})
	}
}

func TestPromHistogramToESBucketOrdering(t *testing.T) {
	metricName := "http_request_duration_seconds_bucket"
	labels := mapstr.M{"runtime": "linux"}

	makeHistogram := func(buckets []*p.Bucket) p.Histogram {
		return p.Histogram{
			SampleCount: proto.Float64(30),
			SampleSum:   proto.Float64(10),
			Bucket:      buckets,
		}
	}

	bucket025 := &p.Bucket{UpperBound: proto.Float64(0.25), CumulativeCount: proto.Float64(10)}
	bucket050 := &p.Bucket{UpperBound: proto.Float64(0.50), CumulativeCount: proto.Float64(20)}
	bucketInf := &p.Bucket{UpperBound: proto.Float64(math.Inf(1)), CumulativeCount: proto.Float64(30)}
	bucket050Low := &p.Bucket{UpperBound: proto.Float64(0.50), CumulativeCount: proto.Float64(5)}

	canonical := makeHistogram([]*p.Bucket{bucket025, bucket050, bucketInf})
	expected := mapstr.M{
		"values": []float64{0.125, 0.375, 0.5},
		"counts": []uint64{0, 0, 0},
	}

	cases := map[string]p.Histogram{
		"canonical order": canonical,
		"shuffled buckets": makeHistogram([]*p.Bucket{
			bucket050,
			bucketInf,
			bucket025,
		}),
		"+Inf first": makeHistogram([]*p.Bucket{
			bucketInf,
			bucket025,
			bucket050,
		}),
		"duplicate bound keeps greatest cumulative": makeHistogram([]*p.Bucket{
			bucket025,
			bucket050Low,
			bucket050,
			bucketInf,
		}),
	}

	for name, histogram := range cases {
		t.Run(name, func(t *testing.T) {
			cache := NewCounterCache(120 * time.Minute)
			result := PromHistogramToES(cache, metricName, labels, &histogram)
			assert.EqualValues(t, expected, result, "output must match canonical ordered input")
		})
	}
}

func TestPromHistogramToESDuplicateBoundRetainsGreatestCumulative(t *testing.T) {
	metricName := "http_request_duration_seconds_bucket"
	labels := mapstr.M{"runtime": "linux"}

	cache := NewCounterCache(120 * time.Minute)
	cache.Start()
	t.Cleanup(cache.Stop)

	firstScrape := p.Histogram{
		Bucket: []*p.Bucket{
			{UpperBound: proto.Float64(0.25), CumulativeCount: proto.Float64(10)},
			{UpperBound: proto.Float64(0.50), CumulativeCount: proto.Float64(20)},
			{UpperBound: proto.Float64(math.Inf(1)), CumulativeCount: proto.Float64(30)},
		},
	}
	firstResult := PromHistogramToES(cache, metricName, labels, &firstScrape)
	assert.EqualValues(t, mapstr.M{
		"values": []float64{0.125, 0.375, 0.5},
		"counts": []uint64{0, 0, 0},
	}, firstResult, "baseline first scrape must seed the counter cache")

	// Duplicate le=0.50 entries appear after sorting; only the greatest cumulative (25) is valid.
	secondScrape := p.Histogram{
		Bucket: []*p.Bucket{
			{UpperBound: proto.Float64(0.50), CumulativeCount: proto.Float64(15)},
			{UpperBound: proto.Float64(0.25), CumulativeCount: proto.Float64(12)},
			{UpperBound: proto.Float64(0.50), CumulativeCount: proto.Float64(25)},
			{UpperBound: proto.Float64(math.Inf(1)), CumulativeCount: proto.Float64(35)},
		},
	}
	secondResult := PromHistogramToES(cache, metricName, labels, &secondScrape)

	// Keeping le=0.50 cumulative 15 would treat the bucket as reset (15 < cached 20) and zero out
	// its rate, yielding counts [2, 0, 3] instead of the correct [2, 3, 0].
	assert.EqualValues(t, mapstr.M{
		"values": []float64{0.125, 0.375, 0.5},
		"counts": []uint64{2, 3, 0},
	}, secondResult, "duplicate-bound dedup must retain the greatest cumulative count on later scrapes")
}

func TestPromHistogramToESDoesNotMutateBuckets(t *testing.T) {
	buckets := []*p.Bucket{
		{UpperBound: proto.Float64(0.50), CumulativeCount: proto.Float64(20)},
		{UpperBound: proto.Float64(math.Inf(1)), CumulativeCount: proto.Float64(30)},
		{UpperBound: proto.Float64(0.25), CumulativeCount: proto.Float64(10)},
	}
	before := make([]float64, len(buckets))
	for i, b := range buckets {
		before[i] = b.GetUpperBound()
	}

	histogram := p.Histogram{Bucket: buckets}
	cache := NewCounterCache(120 * time.Minute)
	_ = PromHistogramToES(cache, "metric_bucket", mapstr.M{}, &histogram)

	for i, b := range buckets {
		assert.Equal(t, before[i], b.GetUpperBound(), "caller bucket slice must not be reordered")
	}
	assert.Equal(t, 0.50, histogram.Bucket[0].GetUpperBound(), "original slice order must be preserved")
}

var normalizedBucketsSink []*p.Bucket

func TestNormalizedHistogramBucketsReusesOrderedUniqueInput(t *testing.T) {
	buckets := []*p.Bucket{
		{UpperBound: proto.Float64(0.25), CumulativeCount: proto.Float64(10)},
		{UpperBound: proto.Float64(0.50), CumulativeCount: proto.Float64(20)},
		{UpperBound: proto.Float64(math.Inf(1)), CumulativeCount: proto.Float64(30)},
	}

	normalized := normalizedHistogramBuckets(buckets)

	assert.Len(t, normalized, len(buckets))
	assert.True(t, &normalized[0] == &buckets[0], "ordered unique input should reuse its backing array")
	assert.Zero(t, testing.AllocsPerRun(100, func() {
		normalizedBucketsSink = normalizedHistogramBuckets(buckets)
	}), "ordered unique input should not allocate")
}
