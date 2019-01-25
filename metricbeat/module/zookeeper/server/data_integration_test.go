// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"github.com/elastic/beats/metricbeat/module/zookeeper"
	"testing"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	t.Skip("Skipping `data.json` generation test")

	f := mbtest.NewReportingMetricSetV2(t, getDataConfig())
	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)

	if err := mbtest.WriteEventsReporterV2(f, t, ""); err != nil {
		t.Fatal("write", err)
	}
}

func getDataConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "zookeeper",
		"metricsets": []string{"server"},
		"hosts":      []string{zookeeper.GetZookeeperEnvHost() + ":" + zookeeper.GetZookeeperEnvPort()},
	}
}
