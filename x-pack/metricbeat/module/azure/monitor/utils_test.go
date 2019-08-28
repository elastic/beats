// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFilterMetrics(t *testing.T) {
	selectedRange := []string{"TotalRequests", "Capacity", "CPUUsage"}
	intersection, difference := filterMetrics(selectedRange, *MockMetricDefinitions())
	assert.Equal(t, intersection, []string{"TotalRequests", "Capacity"})
	assert.Equal(t, difference, []string{"CPUUsage"})
}

func TestFilterAggregations(t *testing.T) {
	selectedRange := []string{"Average", "Minimum"}
	intersection, difference := filterAggregations(selectedRange, *MockMetricDefinitions())
	assert.Equal(t, intersection, []string{"Average"})
	assert.Equal(t, difference, []string{"Minimum"})
}

func TestStringInSlice(t *testing.T) {
	s := "test"
	exists := []string{"hello", "test", "goodbye"}
	noExists := []string{"hello", "goodbye", "notest"}
	result := stringInSlice(s, exists)
	assert.True(t, result)
	result = stringInSlice(s, noExists)
	assert.False(t, result)
}

func TestFilter(t *testing.T) {
	str := []string{"hello", "test", "goodbye", "test"}
	filtered := filter(str)
	assert.Equal(t, len(filtered), 3)
}

func TestIntersections(t *testing.T) {
	firstStr := []string{"test1", "test2", "test2", "test3"}
	sercondStr := []string{"test4", "test5", "test2", "test5", "test3"}
	intersection, difference := intersections(firstStr, sercondStr)
	assert.Equal(t, intersection, []string{"test2", "test3"})
	assert.Equal(t, difference, []string{"test4", "test5"})

	firstStr = []string{"test1", "test2", "test2", "test3"}
	sercondStr = []string{"test4", "test5", "test5"}
	intersection, difference = intersections(firstStr, sercondStr)
	assert.Equal(t, len(intersection), 0)
	assert.Equal(t, difference, []string{"test4", "test5"})

}

func TestGetMetricDefinitionsByNames(t *testing.T) {
	metrics := []string{"TotalRequests", "CPUUsage"}
	result := getMetricDefinitionsByNames(*MockMetricDefinitions(), metrics)
	assert.Equal(t, len(result), 1)
	assert.Equal(t, *result[0].Name.Value, "TotalRequests")
}

func TestExpired(t *testing.T) {
	resConfig := ResourceConfiguration{}
	result := resConfig.expired()
	assert.True(t, result)
}
