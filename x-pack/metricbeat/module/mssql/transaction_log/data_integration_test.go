// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transaction_log

import (
	"testing"

	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"
	mtest "github.com/elastic/beats/v8/x-pack/metricbeat/module/mssql/testing"
)

func TestData(t *testing.T) {
	t.Skip("Skipping `data.json` generation test")

	f := mbtest.NewReportingMetricSetV2(t, mtest.GetConfig("transaction_log"))

	err := mbtest.WriteEventsReporterV2(f, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}
