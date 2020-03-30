// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package test

import (
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/mtest"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"errors"
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


func TestFieldsDocumentation(events []mb.Event, t *testing.T){
	for _, event := range events {
		// RootField
		mtest.CheckEventField("service.name", "string", event, t)
		mtest.CheckEventField("cloud.region", "string", event, t)
		// MetricSetField
		mtest.CheckEventField("empty_receives", "float", event, t)
		mtest.CheckEventField("messages.delayed", "float", event, t)
		mtest.CheckEventField("messages.deleted", "float", event, t)
		mtest.CheckEventField("messages.not_visible", "float", event, t)
		mtest.CheckEventField("messages.received", "float", event, t)
		mtest.CheckEventField("messages.sent", "float", event, t)
		mtest.CheckEventField("messages.visible", "float", event, t)
		mtest.CheckEventField("oldest_message_age.sec", "float", event, t)
		mtest.CheckEventField("sent_message_size", "float", event, t)
		mtest.CheckEventField("queue.name", "string", event, t)
	}
}

// CheckEventField function checks a given field type and compares it with the expected type for integration tests.
func CheckDocumenteded (metricName string, expectedType string, event mb.Event, t *testing.T) {
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
