// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetDimensionValue(t *testing.T) {
	dimensionList := []Dimension{
		{
			Value: "vm1",
			Name:  "VMName",
		},
		{
			Value: "*",
			Name:  "SlotID",
		},
	}
	result := getDimensionValue("VMName", dimensionList)
	assert.Equal(t, result, "vm1")
}

func TestReplaceUpperCase(t *testing.T) {
	result := ReplaceUpperCase("TestReplaceUpper_Case")
	assert.Equal(t, result, "Test_replace_upper_Case")
	// should not split on acronyms
	result = ReplaceUpperCase("CPU_Percentage")
	assert.Equal(t, result, "CPU_Percentage")
}

func TestManagePropertyName(t *testing.T) {
	result := managePropertyName("TestManageProperty_Name")
	assert.Equal(t, result, "test_manage_property_name")

	result = managePropertyName("Test ManageProperty_Name/sec")
	assert.Equal(t, result, "test_manage_property_name_per_sec")

	result = managePropertyName("Test_-_Manage:Property.Name")
	assert.Equal(t, result, "test_manage_property_name")

	result = managePropertyName("Percentage CPU")
	assert.Equal(t, result, "percentage_cpu")
}

func TestMapToKeyValuePoints(t *testing.T) {
	timestamp := time.Now().UTC()
	metricName := "test"
	minValue := 4.0
	maxValue := 42.0
	avgValue := 13.0
	totalValue := 46.0
	countValue := 2.0
	namespace := "test"
	resourceId := "test"
	resourceSubId := "test"
	timeGrain := "PT1M"

	t.Run("test aggregation types", func(t *testing.T) {

		metrics := []Metric{{
			Namespace:     namespace,
			Names:         []string{"test"},
			Aggregations:  "min",
			Values:        []MetricValue{{name: metricName, min: &minValue, timestamp: timestamp}},
			TimeGrain:     timeGrain,
			ResourceId:    resourceId,
			ResourceSubId: resourceSubId,
		}, {
			Namespace:     namespace,
			Names:         []string{"test"},
			Aggregations:  "max",
			Values:        []MetricValue{{name: metricName, max: &maxValue, timestamp: timestamp}},
			TimeGrain:     timeGrain,
			ResourceId:    resourceId,
			ResourceSubId: resourceSubId,
		}, {
			Namespace:     namespace,
			Names:         []string{"test"},
			Aggregations:  "avg",
			Values:        []MetricValue{{name: metricName, avg: &avgValue, timestamp: timestamp}},
			TimeGrain:     timeGrain,
			ResourceId:    resourceId,
			ResourceSubId: resourceSubId,
		}, {
			Namespace:     namespace,
			Names:         []string{"test"},
			Aggregations:  "total",
			Values:        []MetricValue{{name: metricName, total: &totalValue, timestamp: timestamp}},
			TimeGrain:     timeGrain,
			ResourceId:    resourceId,
			ResourceSubId: resourceSubId,
		}, {
			Namespace:     namespace,
			Names:         []string{"test"},
			Aggregations:  "count",
			Values:        []MetricValue{{name: metricName, count: &countValue, timestamp: timestamp}},
			TimeGrain:     timeGrain,
			ResourceId:    resourceId,
			ResourceSubId: resourceSubId,
		}}

		actual := mapToKeyValuePoints(metrics)

		expected := []KeyValuePoint{
			{
				Key:           fmt.Sprintf("%s.%s", metricName, "min"),
				Value:         &minValue,
				Namespace:     namespace,
				TimeGrain:     timeGrain,
				Timestamp:     timestamp,
				ResourceId:    resourceId,
				ResourceSubId: resourceSubId,
				Dimensions:    map[string]interface{}{},
			}, {
				Key:           fmt.Sprintf("%s.%s", metricName, "max"),
				Value:         &maxValue,
				Namespace:     namespace,
				TimeGrain:     timeGrain,
				Timestamp:     timestamp,
				ResourceId:    resourceId,
				ResourceSubId: resourceSubId,
				Dimensions:    map[string]interface{}{},
			}, {
				Key:           fmt.Sprintf("%s.%s", metricName, "avg"),
				Value:         &avgValue,
				Namespace:     namespace,
				TimeGrain:     timeGrain,
				Timestamp:     timestamp,
				ResourceId:    resourceId,
				ResourceSubId: resourceSubId,
				Dimensions:    map[string]interface{}{},
			},
			{
				Key:           fmt.Sprintf("%s.%s", metricName, "total"),
				Value:         &totalValue,
				Namespace:     namespace,
				TimeGrain:     timeGrain,
				Timestamp:     timestamp,
				ResourceId:    resourceId,
				ResourceSubId: resourceSubId,
				Dimensions:    map[string]interface{}{},
			},
			{
				Key:           fmt.Sprintf("%s.%s", metricName, "count"),
				Value:         &countValue,
				Namespace:     namespace,
				TimeGrain:     timeGrain,
				Timestamp:     timestamp,
				ResourceId:    resourceId,
				ResourceSubId: resourceSubId,
				Dimensions:    map[string]interface{}{},
			},
		}

		assert.Equal(t, expected, actual)
	})
}
