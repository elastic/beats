// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package tablespace

import (
	"testing"

	"github.com/elastic/beats/x-pack/metricbeat/module/oracle"

	_ "gopkg.in/goracle.v2"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	//t.Skip("Skip until a proper Docker image is setup for Metricbeat")

	f := mbtest.NewReportingMetricSetV2WithContext(t, getConfig())

	if err := mbtest.WriteEventsReporterV2WithContext(f, t, ""); err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "oracle",
		"metricsets": []string{"tablespace"},
		"hosts":      []string{oracle.GetOracleConnectionDetails()},
		"username":   "sys",
		"password":   "Oradoc_db1",
	}
}
