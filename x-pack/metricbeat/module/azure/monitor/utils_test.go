// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/stretchr/testify/assert"
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

func TestMetricExists(t *testing.T) {
	fl := 12.4
	fl1 := 1.0
	location := time.Location{}
	date1 := time.Date(2019, 12, 12, 12, 12, 12, 12, &location)
	stamp := date.Time{
		Time: date1,
	}
	var name = "Requests"
	insightValue := insights.MetricValue{
		TimeStamp: &stamp,
		Average:   &fl,
		Minimum:   &fl1,
		Maximum:   nil,
		Total:     nil,
		Count:     nil,
	}
	var metricValues = []MetricValue{
		{
			name:      "Requests",
			average:   &fl,
			min:       &fl1,
			max:       nil,
			total:     nil,
			count:     nil,
			timestamp: date1,
		},
		{
			name:      "TotalRequests",
			average:   &fl,
			min:       &fl1,
			max:       nil,
			total:     nil,
			count:     &fl1,
			timestamp: date1,
		},
	}

	result := metricExists(name, insightValue, metricValues)
	assert.True(t, result)
	metricValues[0].name = "TotalRequests"
	result = metricExists(name, insightValue, metricValues)
	assert.False(t, result)
}

func TestMatchMetrics(t *testing.T) {
	prev := Metric{
		resource:     Resource{Name: "vm", Group: "group", ID: "id"},
		namespace:    "namespace",
		names:        []string{"TotalRequests,Capacity"},
		aggregations: "Average,Total",
		dimensions:   []Dimension{{name: "location", value: "West Europe"}},
		values:       nil,
		timeGrain:    "1PM",
	}
	current := Metric{
		resource:     Resource{Name: "vm", Group: "group", ID: "id"},
		namespace:    "namespace",
		names:        []string{"TotalRequests,Capacity"},
		aggregations: "Average,Total",
		dimensions:   []Dimension{{name: "location", value: "West Europe"}},
		values:       []MetricValue{},
		timeGrain:    "1PM",
	}
	result := matchMetrics(prev, current)
	assert.True(t, result)
	current.resource.ID = "id1"
	result = matchMetrics(prev, current)
	assert.False(t, result)
}

func TestMetricIsEmpty(t *testing.T) {
	fl := 12.4
	location := time.Location{}
	stamp := date.Time{
		Time: time.Date(2019, 12, 12, 12, 12, 12, 12, &location),
	}
	insightValue := insights.MetricValue{
		TimeStamp: &stamp,
		Average:   &fl,
		Minimum:   nil,
		Maximum:   nil,
		Total:     nil,
		Count:     nil,
	}
	result := metricIsEmpty(insightValue)
	assert.False(t, result)
	insightValue.Average = nil
	result = metricIsEmpty(insightValue)
	assert.True(t, result)
}

func TestMapResourceGroupFormID(t *testing.T) {
	path := "subscriptions/qw3e45r6t-23ws-1234-6587-1234ed4532/resourceGroups/obs-infrastructure/providers/Microsoft.Compute/virtualMachines/obstestmemleak"
	group := mapResourceGroupFormID(path)
	assert.Equal(t, group, "obs-infrastructure")
}

func TestExpired(t *testing.T) {
	resConfig := ResourceConfiguration{}
	result := resConfig.expired()
	assert.True(t, result)
}
