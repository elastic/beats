// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package collector

import (
	"fmt"
	"math"

	"github.com/elastic/beats/v7/libbeat/common"

	dto "github.com/prometheus/client_model/go"
)

// promHistogramToES takes a Prometheus histogram and converts it to an ES histogram:
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
func promHistogramToES(cc CounterCache, name string, labels common.MapStr, histogram *dto.Histogram) common.MapStr {
	var values []float64
	var counts []uint64

	// calculate centroids and rated counts
	var lastUpper, prevUpper float64
	var sumCount uint64
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

		// take count for this period (rate) + deacumulate
		count := cc.RateUint64(name+labels.String()+fmt.Sprintf("%f", bucket.GetUpperBound()), bucket.GetCumulativeCount()) - sumCount
		counts = append(counts, count)
		sumCount += count
	}

	res := common.MapStr{
		"values": values,
		"counts": counts,
	}

	return res
}
