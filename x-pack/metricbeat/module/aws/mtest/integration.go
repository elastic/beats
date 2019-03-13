// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mtest

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/metricbeat/mb"
)

// GetConfigForTest function gets aws credentials for integration tests.
func GetConfigForTest(metricSetName string, period string) (map[string]interface{}, string) {
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
			"period":            period,
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
