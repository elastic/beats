// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package collector

import (
	"fmt"
	"math"

	p "github.com/elastic/beats/v9/metricbeat/helper/prometheus"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// PromHistogramToES takes a Prometheus histogram and converts it to an ES histogram:
//
// ES histograms look like this:
//
//	"histogram_field" : {
//		   "values" : [0.1, 0.2, 0.3, 0.4, 0.5],
//		   "counts" : [3, 7, 23, 12, 6]
//	}
//
// This code takes a Prometheus histogram and tries to accommodate it into an ES histogram by:
//
//   - calculating centroids for each bucket (values)
//     - for +Inf "le" bucket, use the preceding bucket's value
//     - for the first bucket only: if it has a negative "le", use the value as-is; otherwise use half its value
//     - for all other buckets, use the midpoint from that bucket's value to the preceding bucket's
//   - undoing counters accumulation for each bucket (counts)
//     - `counts` is respresenting an array of rates, where rate of the first bucket is always 0, meaning that it
// 		  was not increased as it is the first
// More details on the histogram transformation logic - https://github.com/elastic/apm-agent-python/pull/1165#discussion_r651397014
//
// https://www.elastic.co/guide/en/elasticsearch/reference/master/histogram.html

func PromHistogramToES(cc CounterCache, name string, labels mapstr.M, histogram *p.Histogram) mapstr.M {
	var values []float64
	var counts []uint64

	// calculate centroids and rated counts
	var lastUpper float64
	var sumCount, prevCount uint64
	for _, bucket := range histogram.GetBucket() {
		// Ignore non-numbers
		if math.IsNaN(bucket.GetCumulativeCount()) || math.IsInf(bucket.GetCumulativeCount(), 0) {
			continue
		}

		bucketUpperBound := bucket.GetUpperBound()
		if bucketUpperBound == math.Inf(0) {
			// Report +Inf bucket as a point, use the preceding bucket's value
			values = append(values, lastUpper)
		} else {
			// for the first bucket only: if it has a negative "le", use the value as-is
			if bucketUpperBound < 0 && len(values) == 0 {
				values = append(values, bucketUpperBound)
			} else {
				// calculate bucket centroid
				values = append(values, lastUpper+(bucketUpperBound-lastUpper)/2.0)
			}
			lastUpper = bucketUpperBound
		}

		// Take count for this period (rate)
		countRate, found := cc.RateUint64(name+labels.String()+fmt.Sprintf("%f", bucketUpperBound), uint64(bucket.GetCumulativeCount()))

		switch {
		case !found:
			// This is a new bucket, consider it zero by now, but still increase the
			// sum to don't deviate following buckets that are not new.
			counts = append(counts, 0)
			sumCount += uint64(bucket.GetCumulativeCount()) - prevCount
		case countRate < sumCount:
			// This should never happen, this means something is wrong in the
			// prometheus response. Handle it to avoid overflowing when deaccumulating.
			counts = append(counts, 0)
		default:
			// Store the deaccumulated count.
			counts = append(counts, countRate-sumCount)
			sumCount = countRate
		}
		prevCount = uint64(bucket.GetCumulativeCount())
	}

	res := mapstr.M{
		"values": values,
		"counts": counts,
	}

	return res
}
