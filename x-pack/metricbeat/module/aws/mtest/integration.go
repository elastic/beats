// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mtest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/metricbeat/mb"
)

// GetConfigForTest function gets aws credentials for integration tests.
func GetConfigForTest(metricSetName string) (map[string]interface{}, string) {
	accessKeyID, okAccessKeyID := os.LookupEnv("AWS_ACCESS_KEY_ID")
	secretAccessKey, okSecretAccessKey := os.LookupEnv("AWS_SECRET_ACCESS_KEY")
	sessionToken, okSessionToken := os.LookupEnv("AWS_SESSION_TOKEN")
	defaultRegion, _ := os.LookupEnv("AWS_REGION")
	if defaultRegion == "" {
		defaultRegion = "us-west-1"
	}

	info := ""
	config := map[string]interface{}{}
	if !okAccessKeyID || accessKeyID == "" {
		info = "Skipping TestFetch; $AWS_ACCESS_KEY_ID not set or set to empty"
	} else if !okSecretAccessKey || secretAccessKey == "" {
		info = "Skipping TestFetch; $AWS_SECRET_ACCESS_KEY not set or set to empty"
	} else {
		config = map[string]interface{}{
			"module":            "aws",
			"period":            "300s",
			"metricsets":        []string{metricSetName},
			"access_key_id":     accessKeyID,
			"secret_access_key": secretAccessKey,
			"default_region":    defaultRegion,
		}

		if okSessionToken && sessionToken != "" {
			config["session_token"] = sessionToken
		}
	}
	return config, info
}

// CheckEventField function checks a given field type and compares it with the expected type for integration tests.
func CheckEventField(metricName string, expectedType string, event mb.Event, t *testing.T) {
	if ok, err := event.MetricSetFields.HasKey(metricName); ok {
		assert.NoError(t, err)
		metricValue, err := event.MetricSetFields.GetValue(metricName)
		assert.NoError(t, err)
		compareType(metricValue, expectedType, t)
	} else if ok, err := event.RootFields.HasKey(metricName); ok {
		assert.NoError(t, err)
		rootValue, err := event.RootFields.GetValue(metricName)
		assert.NoError(t, err)
		compareType(rootValue, expectedType, t)
	}
}

func compareType(metricValue interface{}, expectedType string, t *testing.T) {
	switch metricValue.(type) {
	case float64:
		if expectedType != "float" {
			t.Log("Failed: Field is not in type " + expectedType)
			t.Fail()
		}
	case string:
		if expectedType != "string" {
			t.Log("Failed: Field is not in type " + expectedType)
			t.Fail()
		}
	case int64:
		if expectedType != "int" {
			t.Log("Failed: Field is not in type " + expectedType)
			t.Fail()
		}
	}
	t.Log("Succeed: Field matches type " + expectedType)
}
