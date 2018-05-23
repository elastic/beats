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
	// GetField returns the resulting field name
	GetField() string

	// GetValue returns the resulting value
	GetValue(m *dto.Metric) interface{}
}

// Metric directly maps a Prometheus metric to a Metricbeat field
func Metric(field string) MetricMap {
	return &commonMetric{
		field: field,
	}
}

// KeywordMetric maps a Prometheus metric to a Metricbeat field, stores the
// given keyword when source metric value is 1
func KeywordMetric(field, keyword string) MetricMap {
	return &keywordMetric{
		commonMetric{
			field: field,
		},
		keyword,
	}
}

// BooleanMetric maps a Prometheus metric to a Metricbeat field of bool type
func BooleanMetric(field string) MetricMap {
	return &booleanMetric{
		commonMetric{
			field: field,
		},
	}
}

// LabelMetric maps a Prometheus metric to a Metricbeat field, stores the value
// of a given label on it if the gauge value is 1
func LabelMetric(field, label string) MetricMap {
	return &labelMetric{
		commonMetric{
			field: field,
		},
		label,
	}
}

type commonMetric struct {
	field string
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
		return strings.ToLower(getLabel(metric, m.label))
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
