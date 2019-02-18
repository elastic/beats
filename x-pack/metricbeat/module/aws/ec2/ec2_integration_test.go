// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package ec2

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
			"metricsets":        []string{"ec2"},
			"access_key_id":     accessKeyID,
			"secret_access_key": secretAccessKey,
			"default_region":    defaultRegion,
		}

		if okSessionToken && sessionToken != "" {
			tempCreds["session_token"] = sessionToken
		}

		ec2MetricSet := mbtest.NewReportingMetricSetV2(t, tempCreds)
		events, errs := mbtest.ReportingFetchV2(ec2MetricSet)
		if errs != nil {
			t.Skip("Skipping TestFetch: failed to make api calls. Please check $AWS_ACCESS_KEY_ID, " +
				"$AWS_SECRET_ACCESS_KEY and $AWS_SESSION_TOKEN in config.yml")
		}

		assert.Empty(t, errs)
		if !assert.NotEmpty(t, events) {
			t.FailNow()
		}
		t.Logf("Module: %s Metricset: %s", ec2MetricSet.Module().Name(), ec2MetricSet.Name())

		for _, event := range events {
			// RootField
			checkEventField("service.name", "string", event, t)
			checkEventField("cloud.availability_zone", "string", event, t)
			checkEventField("cloud.provider", "string", event, t)
			checkEventField("cloud.image.id", "string", event, t)
			checkEventField("cloud.instance.id", "string", event, t)
			checkEventField("cloud.machine.type", "string", event, t)
			checkEventField("cloud.provider", "string", event, t)
			checkEventField("cloud.region", "string", event, t)
			// MetricSetField
			checkEventField("cpu.total.pct", "float", event, t)
			checkEventField("cpu.credit_usage", "float", event, t)
			checkEventField("cpu.credit_balance", "float", event, t)
			checkEventField("cpu.surplus_credit_balance", "float", event, t)
			checkEventField("cpu.surplus_credits_charged", "float", event, t)
			checkEventField("network.in.packets", "float", event, t)
			checkEventField("network.out.packets", "float", event, t)
			checkEventField("network.in.bytes", "float", event, t)
			checkEventField("network.out.bytes", "float", event, t)
			checkEventField("diskio.read.bytes", "float", event, t)
			checkEventField("diskio.write.bytes", "float", event, t)
			checkEventField("diskio.read.ops", "float", event, t)
			checkEventField("diskio.write.ops", "float", event, t)
			checkEventField("status.check_failed", "int", event, t)
			checkEventField("status.check_failed_system", "int", event, t)
			checkEventField("status.check_failed_instance", "int", event, t)
		}

		err := mbtest.WriteEventsReporterV2(ec2MetricSet, t, "/")
		if err != nil {
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
