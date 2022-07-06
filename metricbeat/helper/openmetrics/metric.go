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

package openmetrics

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// MetricMap defines the mapping from Openmetrics metric to a Metricbeat field
type MetricMap interface {
	// GetOptions returns the list of metric options
	GetOptions() []MetricOption

	// GetField returns the resulting field name
	GetField() string

	// GetValue returns the resulting value
	GetValue(m *OpenMetric) interface{}
	GetNilValue() interface{}

	// GetConfiguration returns the configuration for the metric
	GetConfiguration() Configuration
}

// Configuration for mappings that needs extended treatment
type Configuration struct {
	// StoreNonMappedLabels indicates if labels found at the metric that are
	// not found at the label map should be part of the resulting event.
	// This setting should be used when the label name is not known beforehand
	StoreNonMappedLabels bool
	// NonMappedLabelsPlacement is used when StoreNonMappedLabels is set to true, and
	// defines the key path at the event under which to store the dynamically found labels.
	// This key path will be added to the events that match this metric along with a subset of
	// key/value pairs will be created under it, one for each non mapped label found.
	//
	// Example:
	//
	// given a metric family in a Openmetrics resource in the form:
	// 		metric1{label1="value1",label2="value2"} 1
	// and not mapping labels but using this entry on a the MetricMap definition:
	// 		"metric1": ExtendedInfoMetric(Configuration{StoreNonMappedLabels: true, NonMappedLabelsPlacement: "mypath"}),
	// would output an event that contains a metricset field as follows
	// 		"mypath": {"label1":"value1","label2":"value2"}
	//
	NonMappedLabelsPlacement string
	// MetricProcessing options are a set of functions that will be
	// applied to metrics after they are retrieved
	MetricProcessingOptions []MetricOption
	// ExtraFields is used to add fields to the
	// event where this metric is included
	ExtraFields mapstr.M
}

// MetricOption adds settings to Metric objects behavior
type MetricOption interface {
	// Process a tuple of field, value and labels from a metric, return the same tuple updated
	Process(field string, value interface{}, labels mapstr.M) (string, interface{}, mapstr.M)
}

// OpFilterMap only processes metrics matching the given filter
func OpFilterMap(label string, filterMap map[string]string) MetricOption {
	return opFilterMap{
		label:     label,
		filterMap: filterMap,
	}
}

// OpLowercaseValue lowercases the value if it's a string
func OpLowercaseValue() MetricOption {
	return opLowercaseValue{}
}

// OpUnixTimestampValue parses a value into a Unix timestamp
func OpUnixTimestampValue() MetricOption {
	return opUnixTimestampValue{}
}

// OpMultiplyBuckets multiplies bucket labels in histograms, useful to change units
func OpMultiplyBuckets(multiplier float64) MetricOption {
	return opMultiplyBuckets{
		multiplier: multiplier,
	}
}

// OpSetSuffix extends the field's name with the given suffix if the value of the metric
// is numeric (and not histogram or quantile), otherwise does nothing
func OpSetNumericMetricSuffix(suffix string) MetricOption {
	return opSetNumericMetricSuffix{
		suffix: suffix,
	}
}

// Metric directly maps a Openmetrics metric to a Metricbeat field
func Metric(field string, options ...MetricOption) MetricMap {
	return &commonMetric{
		field:  field,
		config: Configuration{MetricProcessingOptions: options},
	}
}

// KeywordMetric maps a Openmetrics metric to a Metricbeat field, stores the
// given keyword when source metric value is 1
func KeywordMetric(field, keyword string, options ...MetricOption) MetricMap {
	return &keywordMetric{
		commonMetric{
			field:  field,
			config: Configuration{MetricProcessingOptions: options},
		},
		keyword,
	}
}

// BooleanMetric maps a Openmetrics metric to a Metricbeat field of bool type
func BooleanMetric(field string, options ...MetricOption) MetricMap {
	return &booleanMetric{
		commonMetric{
			field:  field,
			config: Configuration{MetricProcessingOptions: options},
		},
	}
}

// LabelMetric maps a Openmetrics metric to a Metricbeat field, stores the value
// of a given label on it if the gauge value is 1
func LabelMetric(field, label string, options ...MetricOption) MetricMap {
	return &labelMetric{
		commonMetric{
			field:  field,
			config: Configuration{MetricProcessingOptions: options},
		},
		label,
	}
}

// InfoMetric obtains info labels from the given metric and puts them
// into events matching all the key labels present in the metric
func InfoMetric(options ...MetricOption) MetricMap {
	return &infoMetric{
		commonMetric{
			config: Configuration{MetricProcessingOptions: options},
		},
	}
}

// ExtendedInfoMetric obtains info labels from the given metric and puts them
// into events matching all the key labels present in the metric
func ExtendedInfoMetric(configuration Configuration) MetricMap {
	return &infoMetric{
		commonMetric{
			config: configuration,
		},
	}
}

// ExtendedMetric is a metric item that allows extended behaviour
// through configuration
func ExtendedMetric(field string, configuration Configuration) MetricMap {
	return &commonMetric{
		field:  field,
		config: configuration,
	}
}

type commonMetric struct {
	field  string
	config Configuration
}

// GetOptions returns the list of metric options
func (m *commonMetric) GetOptions() []MetricOption {
	return m.config.MetricProcessingOptions
}

// GetField returns the resulting field name
func (m *commonMetric) GetField() string {
	return m.field
}

// GetConfiguration returns the configuration for the metric
func (m *commonMetric) GetConfiguration() Configuration {
	return m.config
}
func (m *commonMetric) GetNilValue() interface{} {
	return nil
}

// GetValue returns the resulting value
func (m *commonMetric) GetValue(metric *OpenMetric) interface{} {
	info := metric.GetInfo()
	if info != nil {
		if info.HasValidValue() {
			return info.GetValue()
		}
	}

	stateset := metric.GetStateset()
	if stateset != nil {
		if stateset.HasValidValue() {
			return stateset.GetValue()
		}
	}

	unknown := metric.GetUnknown()
	if unknown != nil {
		if !math.IsNaN(unknown.GetValue()) && !math.IsInf(unknown.GetValue(), 0) {
			return int64(unknown.GetValue())
		}
	}

	counter := metric.GetCounter()
	if counter != nil {
		if !math.IsNaN(counter.GetValue()) && !math.IsInf(counter.GetValue(), 0) {
			return int64(counter.GetValue())
		}
	}

	gauge := metric.GetGauge()
	if gauge != nil {
		if !math.IsNaN(gauge.GetValue()) && !math.IsInf(gauge.GetValue(), 0) {
			return gauge.GetValue()
		}
	}

	summary := metric.GetSummary()
	if summary != nil {
		value := mapstr.M{}
		if !math.IsNaN(summary.GetSampleSum()) && !math.IsInf(summary.GetSampleSum(), 0) {
			value["sum"] = summary.GetSampleSum()
			value["count"] = summary.GetSampleCount()
		}

		quantiles := summary.GetQuantile()
		percentileMap := mapstr.M{}
		for _, quantile := range quantiles {
			if !math.IsNaN(quantile.GetValue()) && !math.IsInf(quantile.GetValue(), 0) {
				key := strconv.FormatFloat(100*quantile.GetQuantile(), 'f', -1, 64)
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
		value := mapstr.M{}
		if !math.IsNaN(histogram.GetSampleSum()) && !math.IsInf(histogram.GetSampleSum(), 0) {
			value["sum"] = histogram.GetSampleSum()
			value["count"] = histogram.GetSampleCount()
		}

		buckets := histogram.GetBucket()
		bucketMap := mapstr.M{}
		for _, bucket := range buckets {
			if bucket.GetCumulativeCount() != uint64(math.NaN()) && bucket.GetCumulativeCount() != uint64(math.Inf(0)) {
				key := strconv.FormatFloat(bucket.GetUpperBound(), 'f', -1, 64)
				bucketMap[key] = bucket.GetCumulativeCount()
			}
		}

		if len(bucketMap) != 0 {
			value["bucket"] = bucketMap
		}

		return value
	}

	gaugehistogram := metric.GetGaugeHistogram()
	if gaugehistogram != nil {
		value := mapstr.M{}
		if !math.IsNaN(gaugehistogram.GetSampleSum()) && !math.IsInf(gaugehistogram.GetSampleSum(), 0) {
			value["gsum"] = gaugehistogram.GetSampleSum()
			value["gcount"] = gaugehistogram.GetSampleCount()
		}

		buckets := gaugehistogram.GetBucket()
		bucketMap := mapstr.M{}
		for _, bucket := range buckets {
			if bucket.GetCumulativeCount() != uint64(math.NaN()) && bucket.GetCumulativeCount() != uint64(math.Inf(0)) {
				key := strconv.FormatFloat(bucket.GetUpperBound(), 'f', -1, 64)
				bucketMap[key] = bucket.GetCumulativeCount()
			}
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
func (m *keywordMetric) GetValue(metric *OpenMetric) interface{} {
	if gauge := metric.GetGauge(); gauge != nil && gauge.GetValue() == 1 {
		return m.keyword
	}
	return nil
}

type booleanMetric struct {
	commonMetric
}

// GetValue returns the resulting value
func (m *booleanMetric) GetValue(metric *OpenMetric) interface{} {
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
func (m *labelMetric) GetValue(metric *OpenMetric) interface{} {
	if gauge := metric.GetGauge(); gauge != nil && gauge.GetValue() == 1 {
		return getLabel(metric, m.label)
	}
	return nil
}

func getLabel(metric *OpenMetric, name string) string {
	for _, label := range metric.GetLabel() {
		if label.Name == name {
			return label.Value
		}
	}
	return ""
}

type infoMetric struct {
	commonMetric
}

// GetValue returns the resulting value
func (m *infoMetric) GetValue(metric *OpenMetric) interface{} {
	return ""
}

// GetField returns the resulting field name
func (m *infoMetric) GetField() string {
	return ""
}

type opFilterMap struct {
	label     string
	filterMap map[string]string
}

// Called by the Openmetrics helper to apply extra options on retrieved metrics
// Check whether the value of the specified label is allowed and, if yes, return the metric via the specified mapped field
// Else, if the specified label does not match the filter, return nil
// This is useful in cases where multiple Metricbeat fields need to be defined per Openmetrics metric, based on label values
func (o opFilterMap) Process(field string, value interface{}, labels mapstr.M) (string, interface{}, mapstr.M) {
	for k, v := range o.filterMap {
		if labels[o.label] == k {
			return fmt.Sprintf("%v.%v", field, v), value, labels
		}
	}
	return "", nil, nil
}

type opLowercaseValue struct{}

// Process will lowercase the given value if it's a string
func (o opLowercaseValue) Process(field string, value interface{}, labels mapstr.M) (string, interface{}, mapstr.M) {
	if val, ok := value.(string); ok {
		value = strings.ToLower(val)
	}
	return field, value, labels
}

type opMultiplyBuckets struct {
	multiplier float64
}

// Process will multiply the bucket labels if it is an histogram with numeric labels
func (o opMultiplyBuckets) Process(field string, value interface{}, labels mapstr.M) (string, interface{}, mapstr.M) {
	histogram, ok := value.(mapstr.M)
	if !ok {
		return field, value, labels
	}
	bucket, ok := histogram["bucket"].(mapstr.M)
	if !ok {
		return field, value, labels
	}
	sum, ok := histogram["sum"].(float64)
	if !ok {
		return field, value, labels
	}
	multiplied := mapstr.M{}
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

type opSetNumericMetricSuffix struct {
	suffix string
}

// Process will extend the field's name with the given suffix
func (o opSetNumericMetricSuffix) Process(field string, value interface{}, labels mapstr.M) (string, interface{}, mapstr.M) {
	_, ok := value.(float64)
	if !ok {
		return field, value, labels
	}
	field = fmt.Sprintf("%v.%v", field, o.suffix)
	return field, value, labels
}

type opUnixTimestampValue struct {
}

// Process converts a value in seconds into an unix time
func (o opUnixTimestampValue) Process(field string, value interface{}, labels mapstr.M) (string, interface{}, mapstr.M) {
	return field, common.Time(time.Unix(int64(value.(float64)), 0)), labels
}

// OpLabelKeyPrefixRemover removes prefix from label keys
func OpLabelKeyPrefixRemover(prefix string) MetricOption {
	return opLabelKeyPrefixRemover{prefix}
}

// opLabelKeyPrefixRemover is a metric option processor that removes a prefix from the key of a label set
type opLabelKeyPrefixRemover struct {
	Prefix string
}

// Process modifies the labels map, removing a prefix when found at keys of the labels set.
// For each label, if the key is found a new key will be created hosting the same value and the
// old key will be deleted.
// Fields, values and not prefixed labels will remain unmodified.
func (o opLabelKeyPrefixRemover) Process(field string, value interface{}, labels mapstr.M) (string, interface{}, mapstr.M) {
	renameKeys := []string{}
	for k := range labels {
		if len(k) < len(o.Prefix) {
			continue
		}
		if k[:6] == o.Prefix {
			renameKeys = append(renameKeys, k)
		}
	}

	for i := range renameKeys {
		v := labels[renameKeys[i]]
		delete(labels, renameKeys[i])
		labels[renameKeys[i][len(o.Prefix):]] = v
	}
	return "", value, labels
}
