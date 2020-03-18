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

// promHistogramToES takes a Prometheus histogram and converts it to
func promHistogramToES(cc CounterCache, name string, labels common.MapStr, histogram *dto.Histogram) common.MapStr {
	var values []float64
	var counts []uint64

	// calculate centroids and rated counts
	var lastUpper float64
	var sumCount uint64
	for _, bucket := range histogram.GetBucket() {
		// Ignore non-numbers
		if bucket.GetCumulativeCount() == uint64(math.NaN()) || bucket.GetCumulativeCount() == uint64(math.Inf(0)) {
			continue
		}

		if bucket.GetUpperBound() == math.Inf(0) {
			// Report +Inf bucket as a point
			values = append(values, lastUpper)
		} else {
			// calculate bucket centroid
			values = append(values, lastUpper+(bucket.GetUpperBound()-lastUpper)/2.0)
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
