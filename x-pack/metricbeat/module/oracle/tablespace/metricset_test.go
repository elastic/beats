// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package tablespace

import (
	"testing"

	_ "gopkg.in/goracle.v2"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/oracle"
)

func TestData(t *testing.T) {
	t.Skip("Skip until a proper Docker image is setup for Metricbeat")
	r := compose.EnsureUp(t, "oracle")

	f := mbtest.NewReportingMetricSetV2WithContext(t, getConfig(r.Host()))

	if err := mbtest.WriteEventsReporterV2WithContext(f, t, ""); err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "oracle",
		"metricsets": []string{"tablespace"},
		"hosts":      []string{oracle.GetOracleConnectionDetails(host)},
	}
}
