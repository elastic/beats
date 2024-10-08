// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/stretchr/testify/assert"
)

func TestMetricExists(t *testing.T) {
	fl := 12.4
	fl1 := 1.0
	location := time.Location{}
	date1 := time.Date(2019, 12, 12, 12, 12, 12, 12, &location)
	stamp := date.Time{
		Time: date1,
	}
	var name = "Requests"
	insightValue := armmonitor.MetricValue{
		TimeStamp: &stamp.Time,
		Average:   &fl,
		Minimum:   &fl1,
		Maximum:   nil,
		Total:     nil,
		Count:     nil,
	}
	var metricValues = []MetricValue{
		{
			name:      "Requests",
			avg:       &fl,
			min:       &fl1,
			max:       nil,
			total:     nil,
			count:     nil,
			timestamp: date1,
		},
		{
			name:      "TotalRequests",
			avg:       &fl,
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
		ResourceId:   "id",
		Namespace:    "namespace",
		Names:        []string{"TotalRequests,Capacity"},
		Aggregations: "Average,Total",
		Dimensions:   []Dimension{{Name: "location", Value: "West Europe"}},
		Values:       nil,
		TimeGrain:    "1PM",
	}
	current := Metric{
		ResourceId:   "id",
		Namespace:    "namespace",
		Names:        []string{"TotalRequests,Capacity"},
		Aggregations: "Average,Total",
		Dimensions:   []Dimension{{Name: "location", Value: "West Europe"}},
		Values:       []MetricValue{},
		TimeGrain:    "1PM",
	}
	result := matchMetrics(prev, current)
	assert.True(t, result)
	current.ResourceId = "id1"
	result = matchMetrics(prev, current)
	assert.False(t, result)
}

func TestMetricIsEmpty(t *testing.T) {
	fl := 12.4
	location := time.Location{}
	stamp := date.Time{
		Time: time.Date(2019, 12, 12, 12, 12, 12, 12, &location),
	}
	insightValue := armmonitor.MetricValue{
		TimeStamp: &stamp.Time,
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

func TestGetResourceGroupFromID(t *testing.T) {
	path := "subscriptions/qw3e45r6t-23ws-1234-6587-1234ed4532/resourceGroups/obs-infrastructure/providers/Microsoft.Compute/virtualMachines/obstestmemleak"
	group := getResourceGroupFromId(path)
	assert.Equal(t, group, "obs-infrastructure")
}

func TestExpired(t *testing.T) {
	resConfig := ResourceConfiguration{}
	result := resConfig.Expired()
	assert.True(t, result)
}

func TestCompareMetricValues(t *testing.T) {
	var val1 *float64
	var val2 *float64
	result := compareMetricValues(val1, val2)
	assert.True(t, result)
	float1 := 1.23
	val1 = &float1
	result = compareMetricValues(val1, val2)
	assert.False(t, result)
	val2 = &float1
	result = compareMetricValues(val1, val2)
	assert.True(t, result)
}

func TestContainsResource(t *testing.T) {
	resourceId := "resId"
	resourceList := []Resource{
		{
			Name: "resource name",
			Id:   "resId",
		},
		{
			Name: "resource name1",
			Id:   "resId1",
		},
		{
			Name: "resource name2",
			Id:   "resId2",
		},
	}
	ok := containsResource(resourceId, resourceList)
	assert.True(t, ok)
	resourceId = "ressId"
	ok = containsResource(resourceId, resourceList)
	assert.False(t, ok)
}

func TestGetVM(t *testing.T) {
	vmName := "resource name1"
	vmResourceList := []VmResource{
		{
			Name: "resource name",
			Id:   "resId",
		},
		{
			Name: "resource name1",
			Id:   "resId1",
		},
		{
			Name: "resource name2",
			Id:   "resId2",
		},
	}
	vm, ok := getVM(vmName, vmResourceList)
	assert.True(t, ok)
	assert.Equal(t, vm.Name, vmName)
	assert.Equal(t, vm.Id, "resId1")
	vmName = "resource name3"
	vm, ok = getVM(vmName, vmResourceList)
	assert.False(t, ok)
	assert.Equal(t, vm.Name, "")
	assert.Equal(t, vm.Id, "")
}

func TestGetInstanceId(t *testing.T) {
	dimensionValue := "sfjsfjghhbsjsjskjkf"
	result := getInstanceId(dimensionValue)
	assert.Empty(t, result)
	dimensionValue = "fjsfhfhsjhjsfs_34"
	result = getInstanceId(dimensionValue)
	assert.Equal(t, result, "34")
	dimensionValue = "fjsfhfhsjhjsfs_34sjsjfhsfsjjsjf_242"
	result = getInstanceId(dimensionValue)
	assert.Equal(t, result, "242")
}
