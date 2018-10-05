// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package performance

import (
	"testing"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "mssql")

	f := mbtest.NewReportingMetricSetV2(t, getConfig())
	events, errs := mbtest.ReportingFetchV2(f)
	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}

	t.Logf("Module: %s Metricset: %s", f.Module().Name(), f.Name())

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

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "mssql",
		"metricsets": []string{"performance"},
		"host":       "127.0.0.1",
		"user":       "sa",
		"password":   "1234_asdf",
		"port":       1433,
	}
}
