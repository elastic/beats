// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// skipping tests on windows 32 bit versions, not supported
//go:build !integration && !windows && !386
// +build !integration,!windows,!386

package collector

import (
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/common"
)

// TestPromHistogramToES tests that calling PromHistogramToES multiple
// times with the same cache produces each time the expected results.
func TestPromHistogramToES(t *testing.T) {
	type sample struct {
		histogram dto.Histogram
		expected  common.MapStr
	}

	cases := map[string]struct {
		samples []sample
	}{
		"one histogram": {
			samples: []sample{
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(10),
						SampleSum:   proto.Float64(10),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(10),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{0},
						"values": []float64{0.495},
					},
				},
			},
		},
		"two histogram": {
			samples: []sample{
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(10),
						SampleSum:   proto.Float64(10),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(10),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{0},
						"values": []float64{0.495},
					},
				},
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(12),
						SampleSum:   proto.Float64(10.123),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(12),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{2},
						"values": []float64{0.495},
					},
				},
			},
		},
		"new bucket on the go": {
			samples: []sample{
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(10),
						SampleSum:   proto.Float64(10),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(10),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{0},
						"values": []float64{0.495},
					},
				},
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(13),
						SampleSum:   proto.Float64(15.23),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(12),
							},
							// New bucket on the go
							{
								UpperBound:      proto.Float64(9.99),
								CumulativeCount: proto.Uint64(13),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{2, 0},
						"values": []float64{0.495, 5.49},
					},
				},
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(15),
						SampleSum:   proto.Float64(16.33),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(13),
							},
							{
								UpperBound:      proto.Float64(9.99),
								CumulativeCount: proto.Uint64(15),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{1, 1},
						"values": []float64{0.495, 5.49},
					},
				},
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(16),
						SampleSum:   proto.Float64(16.33),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(13),
							},
							{
								UpperBound:      proto.Float64(9.99),
								CumulativeCount: proto.Uint64(16),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{0, 1},
						"values": []float64{0.495, 5.49},
					},
				},
			},
		},
		"new smaller bucket on the go": {
			samples: []sample{
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(10),
						SampleSum:   proto.Float64(10),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(10),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{0},
						"values": []float64{0.495},
					},
				},
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(13),
						SampleSum:   proto.Float64(15.23),
						Bucket: []*dto.Bucket{
							// New bucket on the go
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Uint64(1),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(13),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{0, 2},
						"values": []float64{0.045, 0.54},
					},
				},
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(15),
						SampleSum:   proto.Float64(16.33),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Uint64(2),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(15),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{1, 1},
						"values": []float64{0.045, 0.54},
					},
				},
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(16),
						SampleSum:   proto.Float64(16.33),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Uint64(3),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(16),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{1, 0},
						"values": []float64{0.045, 0.54},
					},
				},
			},
		},
		"new bucket between two other buckets on the go": {
			samples: []sample{
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(10),
						SampleSum:   proto.Float64(10),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Uint64(0),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(10),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{0, 0},
						"values": []float64{0.045, 0.54},
					},
				},
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(13),
						SampleSum:   proto.Float64(15.23),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Uint64(1),
							},
							// New bucket
							{
								UpperBound:      proto.Float64(0.49),
								CumulativeCount: proto.Uint64(2),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(13),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{1, 0, 1},
						"values": []float64{0.045, 0.29000000000000004, 0.74},
					},
				},
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(16),
						SampleSum:   proto.Float64(16.33),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Uint64(2),
							},
							{
								UpperBound:      proto.Float64(0.49),
								CumulativeCount: proto.Uint64(4),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(16),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{1, 1, 1},
						"values": []float64{0.045, 0.29000000000000004, 0.74},
					},
				},
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(18),
						SampleSum:   proto.Float64(16.33),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Uint64(3),
							},
							{
								UpperBound:      proto.Float64(0.49),
								CumulativeCount: proto.Uint64(5),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(18),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{1, 0, 1},
						"values": []float64{0.045, 0.29000000000000004, 0.74},
					},
				},
			},
		},
		"wrong buckets": {
			samples: []sample{
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(10),
						SampleSum:   proto.Float64(10),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Uint64(10),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(8),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{0, 0},
						"values": []float64{0.045, 0.54},
					},
				},
				{
					histogram: dto.Histogram{
						SampleCount: proto.Uint64(12),
						SampleSum:   proto.Float64(10.45),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(0.09),
								CumulativeCount: proto.Uint64(12),
							},
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(8),
							},
						},
					},
					expected: common.MapStr{
						"counts": []uint64{2, 0},
						"values": []float64{0.045, 0.54},
					},
				},
			},
		},
	}

	metricName := "somemetric"
	labels := common.MapStr{}

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
