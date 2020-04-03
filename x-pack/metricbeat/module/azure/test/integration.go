// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package test

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/mb"
)

// GetConfig function gets azure credentials for integration tests.
func GetConfig(t *testing.T, metricSetName string) map[string]interface{} {
	t.Helper()

	clientId, ok := os.LookupEnv("AZURE_CLIENT_ID")
	if !ok {
		t.Fatal("Could not find var AZURE_CLIENT_ID")
	}
	clientSecret, ok := os.LookupEnv("AZURE_CLIENT_SECRET")
	if !ok {
		t.Fatal("Could not find var AZURE_CLIENT_SECRET")
	}
	tenantId, ok := os.LookupEnv("AZURE_TENANT_ID")
	if !ok {
		t.Fatal("Could not find var AZURE_TENANT_ID")
	}
	subId, ok := os.LookupEnv("AZURE_SUBSCRIPTION_ID")
	if !ok {
		t.Fatal("Could not find var AZURE_SUBSCRIPTION_ID")
	}
	return map[string]interface{}{
		"module":                "azure",
		"period":                "300s",
		"refresh_list_interval": "600s",
		"metricsets":            []string{metricSetName},
		"client_id":             clientId,
		"client_secret":         clientSecret,
		"tenant_id":             tenantId,
		"subscription_id":       subId,
	}
}

// TestFieldsDocumentation func checks if all the documented fields have the expected type
func TestFieldsDocumentation(t *testing.T, events []mb.Event) {
	for _, event := range events {
		// RootField
		checkIsDocumented("service.name", "string", event, t)
		checkIsDocumented("cloud.provider", "string", event, t)
		checkIsDocumented("cloud.region", "string", event, t)
		checkIsDocumented("cloud.instance.name", "string", event, t)
		checkIsDocumented("cloud.instance.id", "string", event, t)

		// MetricSetField
		checkIsDocumented("azure.timegrain", "string", event, t)
		checkIsDocumented("azure.subscription_id", "string", event, t)
		checkIsDocumented("azure.namespace", "string", event, t)
		checkIsDocumented("azure.resource.type", "string", event, t)
		checkIsDocumented("azure.resource.group", "string", event, t)
	}
}

// checkIsDocumented function checks a given field type and compares it with the expected type for integration tests.
// this implementation is only temporary, will be replaced by issue https://github.com/elastic/beats/issues/17315
func checkIsDocumented(metricName string, expectedType string, event mb.Event, t *testing.T) {
	t.Helper()

	ok1, err1 := event.MetricSetFields.HasKey(metricName)
	ok2, err2 := event.RootFields.HasKey(metricName)
	if ok1 || ok2 {
		if ok1 {
			assert.NoError(t, err1)
			metricValue, err := event.MetricSetFields.GetValue(metricName)
			assert.NoError(t, err)
			err = compareType(metricValue, expectedType, metricName)
			assert.NoError(t, err)
			t.Log("Succeed: Field " + metricName + " matches type " + expectedType)
		} else if ok2 {
			assert.NoError(t, err2)
			rootValue, err := event.RootFields.GetValue(metricName)
			assert.NoError(t, err)
			err = compareType(rootValue, expectedType, metricName)
			assert.NoError(t, err)
			t.Log("Succeed: Field " + metricName + " matches type " + expectedType)
		}
	} else {
		t.Log("Field " + metricName + " does not exist in metric set fields")
	}
}

func compareType(metricValue interface{}, expectedType string, metricName string) (err error) {
	switch metricValue.(type) {
	case float64:
		if expectedType != "float" {
			err = errors.New("Failed: Field " + metricName + " is not in type " + expectedType)
		}
	case string:
		if expectedType != "string" {
			err = errors.New("Failed: Field " + metricName + " is not in type " + expectedType)
		}
	case int64:
		if expectedType != "int" {
			err = errors.New("Failed: Field " + metricName + " is not in type " + expectedType)
		}
	}
	return
}
