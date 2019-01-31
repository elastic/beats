// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package s3

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	accessKeyID, okAccessKeyID := os.LookupEnv("AWS_ACCESS_KEY_ID")
	secretAccessKey, okSecretAccessKey := os.LookupEnv("AWS_SECRET_ACCESS_KEY")
	sessionToken, okSessionToken := os.LookupEnv("AWS_SESSION_TOKEN")
	defaultRegion, _ := os.LookupEnv("AWS_REGION")

	if !okAccessKeyID || accessKeyID == "" {
		t.Skip("Skipping TestFetch; $AWS_ACCESS_KEY_ID not set or set to empty")
	} else if !okSecretAccessKey || secretAccessKey == "" {
		t.Skip("Skipping TestFetch; $AWS_SECRET_ACCESS_KEY not set or set to empty")
	} else {
		tempCreds := map[string]interface{}{
			"module":            "aws",
			"period":            "300s",
			"metricsets":        []string{"s3"},
			"access_key_id":     accessKeyID,
			"secret_access_key": secretAccessKey,
			"default_region":    defaultRegion,
		}

		if okSessionToken && sessionToken != "" {
			tempCreds["session_token"] = sessionToken
		}

		s3MetricSet := mbtest.NewReportingMetricSetV2(t, tempCreds)
		events, err := mbtest.ReportingFetchV2(s3MetricSet)
		if err != nil {
			t.Skip("Skipping TestFetch: failed to make api calls. Please check $AWS_ACCESS_KEY_ID, " +
				"$AWS_SECRET_ACCESS_KEY and $AWS_SESSION_TOKEN in config.yml")
		}

		assert.Empty(t, err)
		if !assert.NotEmpty(t, events) {
			t.FailNow()
		}

		t.Logf("Module: %s Metricset: %s", s3MetricSet.Module().Name(), s3MetricSet.Name())
		for _, event := range events {
			// RootField
			checkEventField("service.name", "string", event, t)
			checkEventField("cloud.provider", "string", event, t)
			checkEventField("cloud.region", "string", event, t)
			// MetricSetField
			checkEventField("bucket.name", "string", event, t)
			checkEventField("bucket.storage.type", "string", event, t)
			checkEventField("bucket.size.bytes", "float", event, t)
			checkEventField("object.count", "int", event, t)
		}

		errs := mbtest.WriteEventsReporterV2(s3MetricSet, t, "/")
		if errs != nil {
			t.Fatal("write", err)
		}
	}
}

func checkEventField(metricName string, expectedType string, event mb.Event, t *testing.T) {
	if ok, err := event.MetricSetFields.HasKey(metricName); ok {
		assert.NoError(t, err)
		metricValue, err := event.MetricSetFields.GetValue(metricName)
		assert.NoError(t, err)

		switch metricValue.(type) {
		case float64:
			if expectedType != "float" {
				t.Log("Failed: Field " + metricName + " is not in type " + expectedType)
				t.Fail()
			}
		case string:
			if expectedType != "string" {
				t.Log("Failed: Field " + metricName + " is not in type " + expectedType)
				t.Fail()
			}
		case int64:
			if expectedType != "int" {
				t.Log("Failed: Field " + metricName + " is not in type " + expectedType)
				t.Fail()
			}
		}
		t.Log("Succeed: Field " + metricName + " matches type " + expectedType)
	}
}
