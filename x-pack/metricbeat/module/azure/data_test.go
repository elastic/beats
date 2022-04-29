// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestReturnAllDimensions(t *testing.T) {
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
	result, dims := returnAllDimensions(dimensionList)
	assert.True(t, result)
	assert.Equal(t, len(dims), 1)
	assert.Equal(t, dims[0].Name, "SlotID")
	assert.Equal(t, dims[0].Value, "*")
}

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

func TestCreateEvent(t *testing.T) {
	createTime, err := time.Parse(time.RFC3339, "2020-02-28T20:53:03Z")
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	resource := Resource{
		Id:           "resId",
		Name:         "res",
		Location:     "west_europe",
		Type:         "resType",
		Group:        "resGroup",
		Tags:         nil,
		Subscription: "subId",
	}
	metric := Metric{
		ResourceId:   "resId",
		Namespace:    "namespace1",
		Names:        []string{"Percentage CPU"},
		Aggregations: "",
		Dimensions:   nil,
		Values:       nil,
		TimeGrain:    "",
	}
	var total float64 = 23
	metricValues := []MetricValue{
		{
			name:       "Percentage CPU",
			avg:        nil,
			min:        nil,
			max:        nil,
			total:      &total,
			count:      nil,
			timestamp:  time.Time{},
			dimensions: nil,
		},
	}
	event, list := createEvent(createTime, metric, resource, metricValues)
	assert.NotNil(t, event)
	assert.NotNil(t, list)
	assert.Equal(t, event.Timestamp, createTime)
	sub, err := event.ModuleFields.GetValue("subscription_id")
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	assert.Equal(t, sub, resource.Subscription)
	namespace, err := event.ModuleFields.GetValue("namespace")
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	assert.Equal(t, namespace, metric.Namespace)
	val, err := list.GetValue("percentage_cpu")
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	assert.Equal(t, val.(mapstr.M), mapstr.M{"total": total})
}
