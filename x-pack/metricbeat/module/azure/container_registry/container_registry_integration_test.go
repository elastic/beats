// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && azure
// +build integration,azure

package container_registry

import (
	"testing"

	"github.com/menderesk/beats/v7/x-pack/metricbeat/module/azure/test"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"

	// Register input module and metricset
	_ "github.com/menderesk/beats/v7/x-pack/metricbeat/module/azure/monitor"
)

func TestFetchMetricset(t *testing.T) {
	config := test.GetConfig(t, "container_registry")
	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(metricSet)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	mbtest.TestMetricsetFieldsDocumented(t, metricSet, events)
}

func TestData(t *testing.T) {
	config := test.GetConfig(t, "container_registry")
	metricSet := mbtest.NewFetcher(t, config)
	metricSet.WriteEvents(t, "/")
}
