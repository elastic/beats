// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package collector

import (
	"fmt"
	"math"

	dto "github.com/prometheus/client_model/go"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// PromHistogramToES takes a Prometheus histogram and converts it to an ES histogram:
//
// ES histograms look like this:
//
//   "histogram_field" : {
//      "values" : [0.1, 0.2, 0.3, 0.4, 0.5],
//      "counts" : [3, 7, 23, 12, 6]
//   }
//
// This code takes a Prometheus histogram and tries to accomodate it into an ES histogram by:
//  - calculating centroids for each bucket (values)
//  - undoing counters accumulation for each bucket (counts)
//
// https://www.elastic.co/guide/en/elasticsearch/reference/master/histogram.html
func PromHistogramToES(cc CounterCache, name string, labels mapstr.M, histogram *dto.Histogram) mapstr.M {
	var values []float64
	var counts []uint64

	// calculate centroids and rated counts
	var lastUpper, prevUpper float64
	var sumCount, prevCount uint64
	for _, bucket := range histogram.GetBucket() {
		// Ignore non-numbers
		if bucket.GetCumulativeCount() == uint64(math.NaN()) || bucket.GetCumulativeCount() == uint64(math.Inf(0)) {
			continue
		}

		if bucket.GetUpperBound() == math.Inf(0) {
			// Report +Inf bucket as a point, interpolating its value
			values = append(values, lastUpper+(lastUpper-prevUpper))
		} else {
			// calculate bucket centroid
			values = append(values, lastUpper+(bucket.GetUpperBound()-lastUpper)/2.0)
			prevUpper = lastUpper
			lastUpper = bucket.GetUpperBound()
		}

		// Take count for this period (rate)
		countRate, found := cc.RateUint64(name+labels.String()+fmt.Sprintf("%f", bucket.GetUpperBound()), bucket.GetCumulativeCount())

		switch {
		case !found:
			// This is a new bucket, consider it zero by now, but still increase the
			// sum to don't deviate following buckets that are not new.
			counts = append(counts, 0)
			sumCount += bucket.GetCumulativeCount() - prevCount
		case countRate < sumCount:
			// This should never happen, this means something is wrong in the
			// prometheus response. Handle it to avoid overflowing when deaccumulating.
			counts = append(counts, 0)
		default:
			// Store the deaccumulated count.
			counts = append(counts, countRate-sumCount)
			sumCount = countRate
		}
		prevCount = bucket.GetCumulativeCount()
	}

	res := mapstr.M{
		"values": values,
		"counts": counts,
	}

	return res
}
