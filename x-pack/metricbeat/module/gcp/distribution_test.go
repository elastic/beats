// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/api/distribution"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestDistributionHistogramToES(t *testing.T) {
	type sample struct {
		histogram *distribution.Distribution
		expected  mapstr.M
	}

	// Histogram samples copied from:
	// https://cloud.google.com/logging/docs/logs-based-metrics/distribution-metrics

	cases := map[string]struct {
		samples []sample
	}{
		"explicit histogram": {
			samples: []sample{
				{
					histogram: &distribution.Distribution{
						BucketCounts: []int64{0, 0, 0, 6, 1, 1},
						Count:        8,
						BucketOptions: &distribution.Distribution_BucketOptions{
							Options: &distribution.Distribution_BucketOptions_ExplicitBuckets{
								ExplicitBuckets: &distribution.Distribution_BucketOptions_Explicit{
									Bounds: []float64{0, 1, 2, 5, 10, 20},
								},
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{0, 0, 0, 6, 1, 1},
						"values": []float64{0, 1, 2, 5, 10, 20},
					},
				},
			},
		},
		"exponential histogram": {
			samples: []sample{
				{
					histogram: &distribution.Distribution{
						BucketCounts: []int64{0, 0, 3, 1},
						Count:        4,
						BucketOptions: &distribution.Distribution_BucketOptions{
							Options: &distribution.Distribution_BucketOptions_ExponentialBuckets{
								ExponentialBuckets: &distribution.Distribution_BucketOptions_Exponential{
									NumFiniteBuckets: 4,
									Scale:            3,
									GrowthFactor:     2,
								},
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{0, 0, 3, 1},
						"values": []float64{6, 12, 24, 48},
					},
				},
			},
		},
		"linear histogram": {
			samples: []sample{
				{
					histogram: &distribution.Distribution{
						BucketCounts: []int64{0, 1, 2, 0},
						Count:        3,
						BucketOptions: &distribution.Distribution_BucketOptions{
							Options: &distribution.Distribution_BucketOptions_LinearBuckets{
								LinearBuckets: &distribution.Distribution_BucketOptions_Linear{
									NumFiniteBuckets: 4,
									Offset:           5,
									Width:            15,
								},
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{0, 1, 2, 0},
						"values": []float64{20, 35, 50, 65},
					},
				},
			},
		},
		"no histogram": {
			samples: []sample{
				{
					histogram: &distribution.Distribution{
						BucketOptions: &distribution.Distribution_BucketOptions{
							Options: &distribution.Distribution_BucketOptions_LinearBuckets{
								LinearBuckets: &distribution.Distribution_BucketOptions_Linear{
									NumFiniteBuckets: 4,
									Offset:           5,
									Width:            15,
								},
							},
						},
					},
					expected: mapstr.M{
						"counts": []uint64{},
						"values": []float64{},
					},
				},
			},
		},
	}

	for title, c := range cases {
		t.Run(title, func(t *testing.T) {
			for i, s := range c.samples {
				histogram := s.histogram
				t.Logf("#%d: %+v", i, histogram)

				result := DistributionHistogramToES(histogram)
				assert.EqualValues(t, s.expected, result)
			}
		})
	}
}
