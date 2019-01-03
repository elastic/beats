// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package ec2

import (
	"fmt"
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
			"module":                "aws",
			"period":                "300s",
			"metricsets":            []string{"ec2"},
			"aws_access_key_id":     accessKeyID,
			"aws_secret_access_key": secretAccessKey,
			"aws_default_region":    defaultRegion,
		}

		if okSessionToken && sessionToken != "" {
			tempCreds["aws_session_token"] = sessionToken
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

		for _, event := range events {
			// RootField
			CheckRootField("service.name", event, t)
			CheckRootField("cloud.availability_zone", event, t)
			CheckRootField("cloud.provider", event, t)
			CheckRootField("cloud.image.id", event, t)
			CheckRootField("cloud.instance.id", event, t)
			CheckRootField("cloud.machine.type", event, t)
			CheckRootField("cloud.provider", event, t)
			CheckRootField("cloud.region", event, t)
			// MetricSetField
			checkMetricSetField("cpu.total.pct", event, t)
			checkMetricSetField("cpu.credit_usage", event, t)
			checkMetricSetField("cpu.credit_balance", event, t)
			checkMetricSetField("cpu.surplus_credit_balance", event, t)
			checkMetricSetField("cpu.surplus_credits_charged", event, t)
			checkMetricSetField("network.in.packets", event, t)
			checkMetricSetField("network.out.packets", event, t)
			checkMetricSetField("network.in.bytes", event, t)
			checkMetricSetField("network.out.bytes", event, t)
			checkMetricSetField("diskio.read.bytes", event, t)
			checkMetricSetField("diskio.write.bytes", event, t)
			checkMetricSetField("diskio.read.ops", event, t)
			checkMetricSetField("diskio.write.ops", event, t)
			checkMetricSetField("status.check_failed", event, t)
			checkMetricSetField("status.check_failed_system", event, t)
			checkMetricSetField("status.check_failed_instance", event, t)
		}

		err := mbtest.WriteEventsReporterV2(awsMetricSet, t, "")
		if err != nil {
			t.Fatal("write", err)
		}
	}
}

func checkMetricSetField(metricName string, event mb.Event, t *testing.T) {
	if ok, err := event.MetricSetFields.HasKey(metricName); ok {
		assert.NoError(t, err)
		metricValue, err := event.MetricSetFields.GetValue(metricName)
		assert.NoError(t, err)
		if userPercentFloat, ok := metricValue.(float64); !ok {
			fmt.Println("failed: userPercentFloat = ", userPercentFloat)
			t.Fail()
		} else {
			assert.True(t, userPercentFloat >= 0)
			fmt.Println("succeed: userPercentFloat = ", userPercentFloat)
		}
	}
}
