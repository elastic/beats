// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package performance

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	mssqltest "github.com/elastic/beats/x-pack/metricbeat/module/mssql/testing"
)

func TestFetch(t *testing.T) {
	logp.TestingSetup()
	compose.EnsureUp(t, "mssql")

	f := mbtest.NewReportingMetricSetV2(t, mssqltest.GetConfig("performance"))
	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}

	//TODO Each event is a different field but the order is unknown to check
	// for _, event := range events {
	// 	pageSplitsSeconds, err := event.MetricSetFields.GetValue("performance.page_splits_seconds")
	// 	assert.NoError(t, err)
	// 	if pageSplitsSecondsFloat, ok := pageSplitsSeconds.(int64); !ok {
	// 		t.Fail()
	// 	} else {
	// 		assert.True(t, pageSplitsSecondsFloat > 0)
	// 	}
	// }
}
