// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package db

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
	"github.com/menderesk/beats/v7/x-pack/metricbeat/module/syncgateway"
)

func TestData(t *testing.T) {
	mux := syncgateway.CreateTestMuxer()
	server := httptest.NewServer(mux)
	defer server.Close()

	f := mbtest.NewReportingMetricSetV2Error(t, syncgateway.GetConfig([]string{"db"}, server.URL))
	if err := mbtest.WriteEventsReporterV2Error(f, t, ""); err != nil {
		t.Fatal("write", err)
	}
}

func TestFetch(t *testing.T) {
	t.Skip("Skipping test because it seems like 'TestMetricsetFieldsDocumented' function, used here in this test; has some kind of bug")
	mux := syncgateway.CreateTestMuxer()
	server := httptest.NewServer(mux)
	defer server.Close()

	config := syncgateway.GetConfig([]string{"db"}, server.URL)
	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)

	events, errs := mbtest.ReportingFetchV2Error(metricSet)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}

	assert.NotEmpty(t, events)
	mbtest.TestMetricsetFieldsDocumented(t, metricSet, events)
}
