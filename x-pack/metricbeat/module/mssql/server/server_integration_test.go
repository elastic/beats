// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	mssqltest "github.com/elastic/beats/x-pack/metricbeat/module/mssql/testing"
)

func TestFetch(t *testing.T) {
	t.Skip("TODO: The queries in this metricset are not returning any results")

	logp.TestingSetup()
	compose.EnsureUp(t, "mssql")

	f := mbtest.NewReportingMetricSetV2(t, mssqltest.GetConfig("server"))
	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}

	for _, event := range events {
		const key = "services.status"

		status, err := event.MetricSetFields.GetValue(key)
		if err != nil {
			t.Fatal(err)
		}

		statusInt64, ok := status.(int64)
		if !ok {
			t.Fatalf("%v is not an int64, but %T", key, statusInt64)
		}
		assert.True(t, statusInt64 > 0)
	}
}
