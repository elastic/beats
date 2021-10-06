// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mtest

import (
	"errors"
	"os"
	"testing"
)

// GetConfigForTest function gets aws credentials for integration tests.
func GetConfigForTest(t *testing.T, metricSetName string, period string) map[string]interface{} {
	t.Helper()

	accessKeyID, okAccessKeyID := os.LookupEnv("AWS_ACCESS_KEY_ID")
	secretAccessKey, okSecretAccessKey := os.LookupEnv("AWS_SECRET_ACCESS_KEY")
	sessionToken, okSessionToken := os.LookupEnv("AWS_SESSION_TOKEN")
	defaultRegion, _ := os.LookupEnv("AWS_REGION")
	if defaultRegion == "" {
		defaultRegion = "us-west-1"
	}

	config := map[string]interface{}{}
	if !okAccessKeyID || accessKeyID == "" {
		t.Fatal("$AWS_ACCESS_KEY_ID not set or set to empty")
	} else if !okSecretAccessKey || secretAccessKey == "" {
		t.Fatal("$AWS_SECRET_ACCESS_KEY not set or set to empty")
	} else {
		config = map[string]interface{}{
			"module":            "aws",
			"period":            period,
			"metricsets":        []string{metricSetName},
			"access_key_id":     accessKeyID,
			"secret_access_key": secretAccessKey,
			"latency":           "5m",
			// You can specify which region to run test on by using regions variable
			// "regions":           []string{"us-east-1"},
		}

		if okSessionToken && sessionToken != "" {
			config["session_token"] = sessionToken
		}
	}
	return config
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
