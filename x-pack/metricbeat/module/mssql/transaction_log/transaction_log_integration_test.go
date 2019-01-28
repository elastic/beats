// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package transaction_log

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	mtest "github.com/elastic/beats/x-pack/metricbeat/module/mssql/testing"
)

func TestFetch(t *testing.T) {
	logp.TestingSetup()
	compose.EnsureUp(t, "mssql")

	f := mbtest.NewReportingMetricSetV2(t, mtest.GetConfig("transaction_log"))
	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)

	for _, event := range events {
		const key = "space_usage.used.pct"

		usedPercent, err := event.MetricSetFields.GetValue(key)
		if err != nil {
			t.Fatal(err)
		}

		userPercentFloat, ok := usedPercent.(float64)
		if !ok {
			t.Fatalf("%v is not a float64, but %T", key, usedPercent)
		}
		assert.True(t, userPercentFloat > 0)
	}
}
