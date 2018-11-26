// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package bar

import (
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	//"github.com/elastic/beats/libbeat/common"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntegrationBar(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2(t, getConfig())
	events, errs := mbtest.ReportingFetchV2(f)
	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}

	t.Logf("Module: %s Metricset: %s", f.Module().Name(), f.Name())

	for _, event := range events {
		counter, err := event.MetricSetFields.GetValue("counter")
		assert.NoError(t, err, "field not found")

		if counterInt, ok := counter.(int); !ok {
			t.Error("error getting int64")
		} else {
			assert.True(t, counterInt > 0, "count is 0")
		}
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "foo",
		"metricsets": []string{"bar"},
		"hosts":      []string{""},
	}
}
