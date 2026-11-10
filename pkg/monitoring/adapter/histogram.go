// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package adapter

import (
	metrics "github.com/rcrowley/go-metrics"
)

// ClearOnVisitHistogram is the same as a go-metrics StandardHistogram
// except that when you visit it, it calls clear on the underlying
// sample.
type ClearOnVisitHistogram struct {
	sample metrics.Sample
}

// NewClearOnVisitHistogram constructs a new ClearOnVisitHistogram
// from a go-metrics Sample.
func NewClearOnVisitHistogram(s metrics.Sample) *ClearOnVisitHistogram {
	return &ClearOnVisitHistogram{sample: s}
}

// Clear clears the histogram and its sample.
func (h *ClearOnVisitHistogram) Clear() {
	h.sample.Clear()
}

// Count returns the number of samples recorded since the histogram was last
// cleared.
func (h *ClearOnVisitHistogram) Count() int64 {
	return h.sample.Count()
}

// Max returns the maximum value in the sample.
func (h *ClearOnVisitHistogram) Max() int64 {
	return h.sample.Max()
}

// Mean returns the mean of the values in the sample.
func (h *ClearOnVisitHistogram) Mean() float64 {
	return h.sample.Mean()
}

// Min returns the minimum value in the sample.
func (h *ClearOnVisitHistogram) Min() int64 {
	return h.sample.Min()
}

// Percentile returns an arbitrary percentile of the values in the sample.
func (h *ClearOnVisitHistogram) Percentile(p float64) float64 {
	return h.sample.Percentile(p)
}

// Percentiles returns a slice of arbitrary percentiles of the values in the
// sample.
func (h *ClearOnVisitHistogram) Percentiles(ps []float64) []float64 {
	return h.sample.Percentiles(ps)
}

// Sample returns the Sample underlying the histogram.
func (h *ClearOnVisitHistogram) Sample() metrics.Sample {
	return h.sample
}

// Snapshot returns a read-only copy of the histogram.
func (h *ClearOnVisitHistogram) Snapshot() metrics.Histogram {
	return &ClearOnVisitHistogramSnapshot{sample: h.sample.Snapshot().(*metrics.SampleSnapshot)} //nolint:errcheck // codepath depends on panic
}

// StdDev returns the standard deviation of the values in the sample.
func (h *ClearOnVisitHistogram) StdDev() float64 {
	return h.sample.StdDev()
}

// Sum returns the sum in the sample.
func (h *ClearOnVisitHistogram) Sum() int64 {
	return h.sample.Sum()
}

// Update samples a new value.
func (h *ClearOnVisitHistogram) Update(v int64) {
	h.sample.Update(v)
}

// Variance returns the variance of the values in the sample.
func (h *ClearOnVisitHistogram) Variance() float64 {
	return h.sample.Variance()
}

// ClearOnVisitHistogramSnapshot is a read-only copy of another
// Histogram.  This is analogous to a go-metrics HistogramSnapshot
type ClearOnVisitHistogramSnapshot struct {
	sample *metrics.SampleSnapshot
}

// Clear panics.
func (*ClearOnVisitHistogramSnapshot) Clear() {
	panic("Clear called on a HistogramSnapshot")
}

// Count returns the number of samples recorded at the time the snapshot was
// taken.
func (h *ClearOnVisitHistogramSnapshot) Count() int64 {
	return h.sample.Count()
}

// Max returns the maximum value in the sample at the time the snapshot was
// taken.
func (h *ClearOnVisitHistogramSnapshot) Max() int64 {
	return h.sample.Max()
}

// Mean returns the mean of the values in the sample at the time the snapshot
// was taken.
func (h *ClearOnVisitHistogramSnapshot) Mean() float64 {
	return h.sample.Mean()
}

// Min returns the minimum value in the sample at the time the snapshot was
// taken.
func (h *ClearOnVisitHistogramSnapshot) Min() int64 {
	return h.sample.Min()
}

// Percentile returns an arbitrary percentile of values in the sample at the
// time the snapshot was taken.
func (h *ClearOnVisitHistogramSnapshot) Percentile(p float64) float64 {
	return h.sample.Percentile(p)
}

// Percentiles returns a slice of arbitrary percentiles of values in the sample
// at the time the snapshot was taken.
func (h *ClearOnVisitHistogramSnapshot) Percentiles(ps []float64) []float64 {
	return h.sample.Percentiles(ps)
}

// Sample returns the Sample underlying the histogram.
func (h *ClearOnVisitHistogramSnapshot) Sample() metrics.Sample {
	return h.sample
}

// Snapshot returns the snapshot.
func (h *ClearOnVisitHistogramSnapshot) Snapshot() metrics.Histogram {
	return h
}

// StdDev returns the standard deviation of the values in the sample at the
// time the snapshot was taken.
func (h *ClearOnVisitHistogramSnapshot) StdDev() float64 {
	return h.sample.StdDev()
}

// Sum returns the sum in the sample at the time the snapshot was taken.
func (h *ClearOnVisitHistogramSnapshot) Sum() int64 {
	return h.sample.Sum()
}

// Update panics.
func (*ClearOnVisitHistogramSnapshot) Update(int64) {
	panic("Update called on a HistogramSnapshot")
}

// Variance returns the variance of inputs at the time the snapshot was taken.
func (h *ClearOnVisitHistogramSnapshot) Variance() float64 {
	return h.sample.Variance()
}
