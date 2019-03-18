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

package prometheus

import (
	"math"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"

	dto "github.com/prometheus/client_model/go"
)

// MetricMap defines the mapping from Prometheus metric to a Metricbeat field
type MetricMap interface {
	// GetOptions returns the list of metric options
	GetOptions() []MetricOption

	// GetField returns the resulting field name
	GetField() string

	// GetValue returns the resulting value
	GetValue(m *dto.Metric) interface{}
}

// MetricOption adds settings to Metric objects behavior
type MetricOption interface {
	// Process a tuple of field, value and labels from a metric, return the same tuple updated
	Process(field string, value interface{}, labels common.MapStr) (string, interface{}, common.MapStr)
}

// OpFilter only processes metrics matching the given filter
func OpFilter(filter map[string]string) MetricOption {
	return opFilter{
		labels: filter,
	}
}

// OpLowercaseValue lowercases the value if it's a string
func OpLowercaseValue() MetricOption {
	return opLowercaseValue{}
}

// OpMultiplyBuckets multiplies bucket labels in histograms, useful to change units
func OpMultiplyBuckets(multiplier float64) MetricOption {
	return opMultiplyBuckets{
		multiplier: multiplier,
	}
}

// Metric directly maps a Prometheus metric to a Metricbeat field
func Metric(field string, options ...MetricOption) MetricMap {
	return &commonMetric{
		field:   field,
		options: options,
	}
}

// KeywordMetric maps a Prometheus metric to a Metricbeat field, stores the
// given keyword when source metric value is 1
func KeywordMetric(field, keyword string, options ...MetricOption) MetricMap {
	return &keywordMetric{
		commonMetric{
			field:   field,
			options: options,
		},
		keyword,
	}
}

// BooleanMetric maps a Prometheus metric to a Metricbeat field of bool type
func BooleanMetric(field string, options ...MetricOption) MetricMap {
	return &booleanMetric{
		commonMetric{
			field:   field,
			options: options,
		},
	}
}

// LabelMetric maps a Prometheus metric to a Metricbeat field, stores the value
// of a given label on it if the gauge value is 1
func LabelMetric(field, label string, options ...MetricOption) MetricMap {
	return &labelMetric{
		commonMetric{
			field:   field,
			options: options,
		},
		label,
	}
}

// InfoMetric obtains info labels from the given metric and puts them
// into events matching all the key labels present in the metric
func InfoMetric(options ...MetricOption) MetricMap {
	return &infoMetric{
		commonMetric{
			options: options,
		},
	}
}

type commonMetric struct {
	field   string
	options []MetricOption
}

// GetOptions returns the list of metric options
func (m *commonMetric) GetOptions() []MetricOption {
	return m.options
}

// GetField returns the resulting field name
func (m *commonMetric) GetField() string {
	return m.field
}

// GetValue returns the resulting value
func (m *commonMetric) GetValue(metric *dto.Metric) interface{} {
	counter := metric.GetCounter()
	if counter != nil {
		return int64(counter.GetValue())
	}

	gauge := metric.GetGauge()
	if gauge != nil {
		return gauge.GetValue()
	}

	summary := metric.GetSummary()
	if summary != nil {
		value := common.MapStr{}
		value["sum"] = summary.GetSampleSum()
		value["count"] = summary.GetSampleCount()

		quantiles := summary.GetQuantile()
		percentileMap := common.MapStr{}
		for _, quantile := range quantiles {
			if !math.IsNaN(quantile.GetValue()) {
				key := strconv.FormatFloat((100 * quantile.GetQuantile()), 'f', -1, 64)
				percentileMap[key] = quantile.GetValue()
			}

		}

		if len(percentileMap) != 0 {
			value["percentile"] = percentileMap
		}

		return value
	}

	histogram := metric.GetHistogram()
	if histogram != nil {
		value := common.MapStr{}
		value["sum"] = histogram.GetSampleSum()
		value["count"] = histogram.GetSampleCount()

		buckets := histogram.GetBucket()
		bucketMap := common.MapStr{}
		for _, bucket := range buckets {
			key := strconv.FormatFloat(bucket.GetUpperBound(), 'f', -1, 64)
			bucketMap[key] = bucket.GetCumulativeCount()
		}

		if len(bucketMap) != 0 {
			value["bucket"] = bucketMap
		}

		return value
	}

	// Other types are not supported here
	return nil
}

type keywordMetric struct {
	commonMetric
	keyword string
}

// GetValue returns the resulting value
func (m *keywordMetric) GetValue(metric *dto.Metric) interface{} {
	if gauge := metric.GetGauge(); gauge != nil && gauge.GetValue() == 1 {
		return m.keyword
	}
	return nil
}

type booleanMetric struct {
	commonMetric
}

// GetValue returns the resulting value
func (m *booleanMetric) GetValue(metric *dto.Metric) interface{} {
	if gauge := metric.GetGauge(); gauge != nil {
		return gauge.GetValue() == 1
	}
	return nil
}

type labelMetric struct {
	commonMetric
	label string
}

// GetValue returns the resulting value
func (m *labelMetric) GetValue(metric *dto.Metric) interface{} {
	if gauge := metric.GetGauge(); gauge != nil && gauge.GetValue() == 1 {
		return getLabel(metric, m.label)
	}
	return nil
}

func getLabel(metric *dto.Metric, name string) string {
	for _, label := range metric.GetLabel() {
		if label.GetName() == name {
			return label.GetValue()
		}
	}
	return ""
}

type infoMetric struct {
	commonMetric
}

// GetValue returns the resulting value
func (m *infoMetric) GetValue(metric *dto.Metric) interface{} {
	return ""
}

// GetField returns the resulting field name
func (m *infoMetric) GetField() string {
	return ""
}

type opFilter struct {
	labels map[string]string
}

// Process will return nil if labels don't match the filter
func (o opFilter) Process(field string, value interface{}, labels common.MapStr) (string, interface{}, common.MapStr) {
	for k, v := range o.labels {
		if labels[k] != v {
			return "", nil, nil
		}
	}
	return field, value, labels
}

type opLowercaseValue struct{}

// Process will lowercase the given value if it's a string
func (o opLowercaseValue) Process(field string, value interface{}, labels common.MapStr) (string, interface{}, common.MapStr) {
	if val, ok := value.(string); ok {
		value = strings.ToLower(val)
	}
	return field, value, labels
}

type opMultiplyBuckets struct {
	multiplier float64
}

// Process will multiply the bucket labels if it is an histogram with numeric labels
func (o opMultiplyBuckets) Process(field string, value interface{}, labels common.MapStr) (string, interface{}, common.MapStr) {
	histogram, ok := value.(common.MapStr)
	if !ok {
		return field, value, labels
	}
	bucket, ok := histogram["bucket"].(common.MapStr)
	if !ok {
		return field, value, labels
	}
	sum, ok := histogram["sum"].(float64)
	if !ok {
		return field, value, labels
	}
	multiplied := common.MapStr{}
	for k, v := range bucket {
		if f, err := strconv.ParseFloat(k, 64); err == nil {
			key := strconv.FormatFloat(f*o.multiplier, 'f', -1, 64)
			multiplied[key] = v
		} else {
			multiplied[k] = v
		}
	}
	histogram["bucket"] = multiplied
	histogram["sum"] = sum * o.multiplier
	return field, histogram, labels
}
