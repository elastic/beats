// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package s3

import (
	"fmt"
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

	accessKeyID = "ASIAZENKQPPNXCVBFA4K"
	secretAccessKey = "afEa6uyRYr+8SnHC98KfrrCdx+Hk1hqBsx3A7jcQ"
	sessionToken = "FQoGZXIvYXdzED0aDLu86SIJdMiFaWMa9yKwATFVfGZ2yjos4nyU6UUHmNn9JWMy8Fw2nkw1PqpEYIu2IYVVdEn905qdFYY2z50pPlsFLAWu8lWzLbx7kwfi2iBKeu2oau9/IDrweDMYfF3UApZybXLvIMHad2pv77MjUbLWIjc8ZIcLHuTYpuofQRXsXJ4JzxDbRbOOOrm9eLDisGGSpzR2QDZDOjHqUM19SsHu568Tl/XYvBrNEPZcGYhAXIXCBzpGWdESaDeQJPvHKPDXyeIF"

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
			fmt.Println("err = ", err)
			t.Skip("Skipping TestFetch: failed to make api calls. Please check $AWS_ACCESS_KEY_ID, " +
				"$AWS_SECRET_ACCESS_KEY and $AWS_SESSION_TOKEN in config.yml")
		}

		assert.Empty(t, err)
		if !assert.NotEmpty(t, events) {
			t.FailNow()
		}
		fmt.Println("events = ", events)
		t.Logf("Module: %s Metricset: %s", s3MetricSet.Module().Name(), s3MetricSet.Name())
	}
}
