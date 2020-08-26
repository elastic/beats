package mtest

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
)

// MetricSet from stats extension
type MetricSet struct {
	mb.ReportingMetricSetV2Error
	usageLastCollectedOn time.Time
	usageNextCollectOn   time.Time
}

// NewMetricSet create a new instance of the MetricSet
func NewMetricSet(base mb.ReportingMetricSetV2Error) (MetricSet, error) {
	ms, ok := base.(MetricSet)

	if !ok {
		const errorMsg = "Not a valid MetricSet"
		return ms, fmt.Errorf(errorMsg)
	}

	return ms, nil
}

// SetUsageLastCollectedOn Set the value for usageLastCollectedOn
func SetUsageLastCollectedOn(metricset MetricSet, t time.Time) {
	metricset.usageLastCollectedOn = t
}

// GetUsageLastCollectedOn Get the value for usageLastCollectedOn
func GetUsageLastCollectedOn(metricset MetricSet) time.Time {
	return metricset.usageLastCollectedOn
}

// SetUsageNextCollectOn Set the value for usageNextCollectOn
func SetUsageNextCollectOn(metricset MetricSet, t time.Time) {
	metricset.usageNextCollectOn = t
}

// GetUsageNextCollectOn Get the value for usageNextCollectOn
func GetUsageNextCollectOn(metricset MetricSet) time.Time {
	return metricset.usageNextCollectOn
}
