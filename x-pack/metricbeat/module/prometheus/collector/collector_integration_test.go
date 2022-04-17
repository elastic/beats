// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration
// +build integration

package collector

import (
	"testing"
	"time"

	"github.com/menderesk/beats/v7/libbeat/tests/compose"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "prometheus")

	config := map[string]interface{}{
		"module":        "prometheus",
		"metricsets":    []string{"collector"},
		"hosts":         []string{service.Host()},
		"use_types":     true,
		"rate_counters": true,
	}
	ms := mbtest.NewReportingMetricSetV2Error(t, config)
	var err error
	for retries := 0; retries < 3; retries++ {
		err = mbtest.WriteEventsReporterV2Error(ms, t, "")
		if err == nil {
			return
		}
		time.Sleep(10 * time.Second)
	}
	t.Fatal("write", err)
}
