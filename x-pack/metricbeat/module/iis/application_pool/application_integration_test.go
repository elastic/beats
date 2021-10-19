// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && windows
// +build integration,windows

package application_pool

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/iis/test"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	if err := test.EnsureIISIsRunning(); err != nil {
		t.Skip("Skipping TestFetch: " + err.Error())
	}
	m := mbtest.NewFetcher(t, test.GetConfig("application_pool"))
	events, errs := m.FetchEvents()
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	if events != nil {
		assert.NotEmpty(t, events)
	}

}

func TestData(t *testing.T) {
	if err := test.EnsureIISIsRunning(); err != nil {
		t.Skip("Skipping TestFetch: " + err.Error())
	}
	metricSet := mbtest.NewReportingMetricSetV2Error(t, test.GetConfig("application_pool"))
	if err := mbtest.WriteEventsReporterV2Error(metricSet, t, "/"); err != nil {
		t.Fatal("write", err)
	}
}
