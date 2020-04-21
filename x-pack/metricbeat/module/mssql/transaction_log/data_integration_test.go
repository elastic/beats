// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transaction_log

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/tests/compose"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	mtest "github.com/elastic/beats/v7/x-pack/metricbeat/module/mssql/testing"
)

func TestData(t *testing.T) {
	logp.TestingSetup()
	service := compose.EnsureUp(t, "mssql")

	f := mbtest.NewReportingMetricSetV2(t, mtest.GetConfig(service.Host(), "transaction_log"))

	err := mbtest.WriteEventsReporterV2(f, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}
