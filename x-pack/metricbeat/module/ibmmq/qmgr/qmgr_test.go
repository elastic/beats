// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package tablespace

import (
	"testing"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	t.Skip("Skip until a proper Docker image is setup for Metricbeat")
	r := compose.EnsureUp(t, "ibmmq")

	f := mbtest.NewReportingMetricSetV2WithContext(t, getConfig(r.Host()))

	if err := mbtest.WriteEventsReporterV2WithContext(f, t, ""); err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(host string) map[string]interface{} {
	cc := map[sting]interface{}{
		"clientMode": "true",
		"mqServer": "DEV.ADMIN.SVRCONN/TCP/127.0.0.1(1414)",
		"user": "admin",
		"password": "passw0rd",
	}
	return map[string]interface{}{
		"module":              "ibmmq",
		"metricsets":          []string{"qmgr"},
		"hosts":               []string{"localhost"},
		"bindingQueueManager": "QM1",
		"cc":                  cc,
	}
}
