// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transaction_log

import (
	"testing"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/x-pack/metricbeat/module/mssql/mtest"
)

func testData(t *testing.T, r compose.R) {
	t.Skip("Skipping `data.json` generation test")

	f := mbtest.NewReportingMetricSetV2(t, mtest.GetConfig(r.Host(), "transaction_log"))
	err := mbtest.WriteEventsReporterV2(f, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}
