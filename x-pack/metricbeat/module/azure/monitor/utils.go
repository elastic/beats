// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"strings"
	"time"
)

// filterMetrics will filter out any unsupported metrics based on the namespace selected
func filterMetrics(selectedRange []string, allRange []insights.MetricDefinition) ([]string, []string) {
	var inRange []string
	var notInRange []string
	var allMetrics []string
	for _, definition := range allRange {
		allMetrics = append(allMetrics, *definition.Name.Value)
	}
	for _, name := range selectedRange {
		if stringInSlice(name, allMetrics) {
			inRange = append(inRange, name)
		} else {
			notInRange = append(notInRange, name)
		}

	}
	return inRange, notInRange
}

// filterAggregations will filter out any unsupported aggregations based on the metrics selected
func filterAggregations(selectedRange []string, metrics []insights.MetricDefinition) ([]string, []string) {
	var difference []string
	var supported = []string{"Average", "Maximum", "Minimum", "Count", "Total"}

	for _, metric := range metrics {
		var metricSupported []string
		for _, agg := range *metric.SupportedAggregationTypes {
			metricSupported = append(metricSupported, string(agg))
		}
		supported, _ = intersections(metricSupported, supported)
	}
	if len(selectedRange) != 0 {
		supported, difference = intersections(supported, selectedRange)
	}
	return supported, difference
}

// stringInSlice is a helper method, will check if string is part of a slice
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// filter is a helper method, will filter out strings not part of a slice
func filter(src []string) (res []string) {
	for _, s := range src {
		newStr := strings.Join(res, " ")
		if !strings.Contains(newStr, s) {
			res = append(res, s)
		}
	}
	return
}

// intersections is a helper method, will compare 2 slices and return their intersection and difference records
func intersections(supported, selected []string) ([]string, []string) {
	var intersection []string
	var difference []string
	str1 := strings.Join(filter(supported), " ")
	for _, s := range filter(selected) {
		if strings.Contains(str1, s) {
			intersection = append(intersection, s)
		} else {
			difference = append(difference, s)
		}
	}
	return intersection, difference
}

// getMetricDefinitionsByNames is a helper method, will compare 2 slices and return their intersection
func getMetricDefinitionsByNames(metricDefs []insights.MetricDefinition, names []string) []insights.MetricDefinition {
	var metrics []insights.MetricDefinition
	for _, def := range metricDefs {
		for _, supportedName := range names {
			if *def.Name.Value == supportedName {
				metrics = append(metrics, def)
			}
		}
	}
	return metrics
}

// expired will check for an expiration time and assign a new one
func (p *ResourceConfiguration) expired() bool {
	if p.refreshInterval <= 0 {
		return true
	}
	p.lastUpdate.Lock()
	defer p.lastUpdate.Unlock()
	if p.lastUpdate.Add(p.refreshInterval).After(time.Now()) {
		return false
	}
	p.lastUpdate.Time = time.Now()
	return true
}
