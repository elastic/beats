// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows
// +build windows

package application_pool

import (
	"testing"
	"time"

	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
)

func TestMetricsetNoErrors(t *testing.T) {
	config := map[string]interface{}{
		"module":     "iis",
		"metricsets": []string{"application_pool"},
	}

	ms := mbtest.NewReportingMetricSetV2Error(t, config)
	mbtest.ReportingFetchV2Error(ms)
	time.Sleep(60 * time.Millisecond)

	_, errs := mbtest.ReportingFetchV2Error(ms)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
}
