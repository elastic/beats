// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && windows
// +build integration,windows

package webserver

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/x-pack/metricbeat/module/iis/test"

	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"

	// Register input module and metricset
	_ "github.com/elastic/beats/v8/metricbeat/module/windows"
	_ "github.com/elastic/beats/v8/metricbeat/module/windows/perfmon"
)

func TestFetch(t *testing.T) {
	if err := test.EnsureIISIsRunning(); err != nil {
		t.Skip("Skipping TestFetch: " + err.Error())
	}
	m := mbtest.NewFetcher(t, test.GetConfig("webserver"))
	events, errs := m.FetchEvents()
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
}

func TestData(t *testing.T) {
	if err := test.EnsureIISIsRunning(); err != nil {
		t.Skip("Skipping TestFetch: " + err.Error())
	}
	metricSet := mbtest.NewReportingMetricSetV2Error(t, test.GetConfig("webserver"))
	if err := mbtest.WriteEventsReporterV2Error(metricSet, t, "/"); err != nil {
		t.Fatal("write", err)
	}
}
