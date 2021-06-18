// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package db

import (
	"net/http/httptest"
	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/syncgateway"
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
