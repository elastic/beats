// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"math"

	"google.golang.org/genproto/googleapis/api/distribution"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func DistributionMetricHasHistogram(d *distribution.Distribution) bool {
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

func DistributionHistogramToES(d *distribution.Distribution) mapstr.M {
	values := []float64{}
	counts := []uint64{}

	if DistributionMetricHasHistogram(d) {
		switch {
		case d.BucketOptions.GetExplicitBuckets() != nil:
			bucket := d.BucketOptions.GetExplicitBuckets()

			for i := 0; i < len(d.BucketCounts); i++ {
				upperBound := bucket.Bounds[i]

				values = append(values, upperBound)
			}
		case d.BucketOptions.GetExponentialBuckets() != nil:
			bucket := d.BucketOptions.GetExponentialBuckets()

			for i := 1; i < len(d.BucketCounts)+1; i++ {
				upperBound := bucket.Scale * (math.Pow(bucket.GrowthFactor, float64(i)))
				values = append(values, upperBound)
			}

		case d.BucketOptions.GetLinearBuckets() != nil:
			bucket := d.BucketOptions.GetLinearBuckets()

			for i := 1; i < len(d.BucketCounts)+1; i++ {
				upperBound := bucket.Offset + (bucket.Width * float64(i))

				values = append(values, upperBound)
			}
		}

		for i := range d.BucketCounts {
			counts = append(counts, uint64(d.BucketCounts[i]))
		}
	}

	res := mapstr.M{
		"values": values,
		"counts": counts,
	}

	return res
}
