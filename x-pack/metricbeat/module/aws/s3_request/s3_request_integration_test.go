// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package s3_request_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
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
			"metricsets":        []string{"s3_request"},
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
			fmt.Println("err = ", err)
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
			aws.CheckEventField("service.name", "string", event, t)
			aws.CheckEventField("cloud.provider", "string", event, t)
			aws.CheckEventField("cloud.region", "string", event, t)
			// MetricSetField
			aws.CheckEventField("bucket.name", "string", event, t)
		}

		errs := mbtest.WriteEventsReporterV2(s3MetricSet, t, "/")
		if errs != nil {
			t.Fatal("write", err)
		}
	}
}
