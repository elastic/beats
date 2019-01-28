// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package s3

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

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

		awsMetricSet := mbtest.NewReportingMetricSetV2(t, tempCreds)
		events, errs := mbtest.ReportingFetchV2(awsMetricSet)
		if errs != nil {
			t.Skip("Skipping TestFetch: failed to make api calls. Please check $AWS_ACCESS_KEY_ID, " +
				"$AWS_SECRET_ACCESS_KEY and $AWS_SESSION_TOKEN in config.yml")
		}

		assert.Empty(t, errs)
		if !assert.NotEmpty(t, events) {
			t.FailNow()
		}
		t.Logf("Module: %s Metricset: %s", awsMetricSet.Module().Name(), awsMetricSet.Name())
	}
}
