// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"math"

	"google.golang.org/genproto/googleapis/api/distribution"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func containsHistogram(d *distribution.Distribution) bool {
	if d.Count == 0 || d.BucketOptions == nil || len(d.BucketCounts) == 0 {
		return false
	}

	var sum int64

	for _, v := range d.BucketCounts {
		sum += v
	}

	// Count must be equal to the sum of values in BucketCounts.
	return d.Count == sum
}

func calcExplicitUpperBound(bucket *distribution.Distribution_BucketOptions_Explicit, i int) float64 {
	return bucket.Bounds[i]
}

func calcExponentialUpperBound(bucket *distribution.Distribution_BucketOptions_Exponential, i int) float64 {
	return bucket.Scale * (math.Pow(bucket.GrowthFactor, float64(i)))
}

func calcLinearUpperBound(bucket *distribution.Distribution_BucketOptions_Linear, i int) float64 {
	return bucket.Offset + (bucket.Width * float64(i))
}

func createHistogram(values []float64, counts []uint64) mapstr.M {
	return mapstr.M{
		"values": values,
		"counts": counts,
	}
}

func DistributionHistogramToES(d *distribution.Distribution) mapstr.M {
	if !containsHistogram(d) {
		return createHistogram([]float64{}, []uint64{})
	}

	values := make([]float64, 0, len(d.BucketCounts))
	counts := make([]uint64, 0, len(d.BucketCounts))

	switch {
	case d.BucketOptions.GetExplicitBuckets() != nil:
		bucket := d.BucketOptions.GetExplicitBuckets()

		for i := range d.BucketCounts {
			values = append(values, calcExplicitUpperBound(bucket, i))
		}
	case d.BucketOptions.GetExponentialBuckets() != nil:
		bucket := d.BucketOptions.GetExponentialBuckets()

		for i := range d.BucketCounts {
			values = append(values, calcExponentialUpperBound(bucket, i+1))
		}
	case d.BucketOptions.GetLinearBuckets() != nil:
		bucket := d.BucketOptions.GetLinearBuckets()

		for i := range d.BucketCounts {
			values = append(values, calcLinearUpperBound(bucket, i+1))
		}
	}

	for i := range d.BucketCounts {
		counts = append(counts, uint64(d.BucketCounts[i]))
	}

	return createHistogram(values, counts)
}
