// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package db

import (
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/elastic/beats/libbeat/tests/compose"
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

	for _, event := range events {
		userPercent, err := event.MetricSetFields.GetValue("log_space_usage.used_percent")
		assert.NoError(t, err)
		if userPercentFloat, ok := userPercent.(float64); !ok {
			t.Fail()
		} else {
			assert.True(t, userPercentFloat > 0)
		}
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "mssql",
		"metricsets": []string{"db"},
		"host":       "127.0.0.1",
		"user":       "sa",
		"password":   "1234_asdf",
		"port":       1433,
	}
}
