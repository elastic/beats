// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"testing"

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
