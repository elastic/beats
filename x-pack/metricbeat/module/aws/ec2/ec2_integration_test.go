// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws/mtest"
)

func TestFetch(t *testing.T) {
	config, info := mtest.GetConfigForTest("ec2")
	if info != "" {
		t.Skip("Skipping TestFetch: " + info)
	}

	ec2MetricSet := mbtest.NewReportingMetricSetV2(t, config)
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
		mtest.CheckEventField("service.name", "string", event, t)
		mtest.CheckEventField("cloud.availability_zone", "string", event, t)
		mtest.CheckEventField("cloud.provider", "string", event, t)
		mtest.CheckEventField("cloud.instance.id", "string", event, t)
		mtest.CheckEventField("cloud.machine.type", "string", event, t)
		mtest.CheckEventField("cloud.provider", "string", event, t)
		mtest.CheckEventField("cloud.region", "string", event, t)
		mtest.CheckEventField("instance.image.id", "string", event, t)
		mtest.CheckEventField("instance.state.name", "string", event, t)
		mtest.CheckEventField("instance.state.code", "int", event, t)
		mtest.CheckEventField("instance.monitoring.state", "string", event, t)
		mtest.CheckEventField("instance.core.count", "int", event, t)
		mtest.CheckEventField("instance.threads_per_core", "int", event, t)

		// MetricSetField
		mtest.CheckEventField("cpu.total.pct", "float", event, t)
		mtest.CheckEventField("cpu.credit_usage", "float", event, t)
		mtest.CheckEventField("cpu.credit_balance", "float", event, t)
		mtest.CheckEventField("cpu.surplus_credit_balance", "float", event, t)
		mtest.CheckEventField("cpu.surplus_credits_charged", "float", event, t)
		mtest.CheckEventField("network.in.packets", "float", event, t)
		mtest.CheckEventField("network.out.packets", "float", event, t)
		mtest.CheckEventField("network.in.bytes", "float", event, t)
		mtest.CheckEventField("network.out.bytes", "float", event, t)
		mtest.CheckEventField("diskio.read.bytes", "float", event, t)
		mtest.CheckEventField("diskio.write.bytes", "float", event, t)
		mtest.CheckEventField("diskio.read.ops", "float", event, t)
		mtest.CheckEventField("diskio.write.ops", "float", event, t)
		mtest.CheckEventField("status.check_failed", "int", event, t)
		mtest.CheckEventField("status.check_failed_system", "int", event, t)
		mtest.CheckEventField("status.check_failed_instance", "int", event, t)
	}
}

func TestData(t *testing.T) {
	config, info := mtest.GetConfigForTest("ec2")
	if info != "" {
		t.Skip("Skipping TestData: " + info)
	}

	ec2MetricSet := mbtest.NewReportingMetricSetV2(t, config)
	errs := mbtest.WriteEventsReporterV2(ec2MetricSet, t, "/")
	if errs != nil {
		t.Fatal("write", errs)
	}
}
