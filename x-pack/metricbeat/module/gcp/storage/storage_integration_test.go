// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && gcp
// +build integration,gcp

package storage

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp/metrics"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestFetch(t *testing.T) {
	config := metrics.GetConfigForTest(t, "storage")
	fmt.Printf("%+v\n", config)

	metricSet := mbtest.NewReportingMetricSetV2WithContext(t, config)
	events, errs := mbtest.ReportingFetchV2WithContext(metricSet)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}

	assert.NotEmpty(t, events)
	mbtest.TestMetricsetFieldsDocumented(t, metricSet, events)
}

func TestData(t *testing.T) {
	metricPrefixIs := func(metricPrefix string) func(e mapstr.M) bool {
		return func(e mapstr.M) bool {
			v, err := e.GetValue(metricPrefix)
			return err == nil && v != nil
		}
	}

	dataFiles := []struct {
		metricPrefix string
		path         string
	}{
		{"gcp.storage", "./_meta/data.json"},
		{"gcp.storage.authz", "./_meta/data_authz.json"},
		{"gcp.storage.network", "./_meta/data_network.json"},
		{"gcp.storage.storage", "./_meta/data_storage.json"},
	}

	config := metrics.GetConfigForTest(t, "storage")

	for _, df := range dataFiles {
		metricSet := mbtest.NewFetcher(t, config)
		t.Run(fmt.Sprintf("metric prefix: %s", df.metricPrefix), func(t *testing.T) {
			metricSet.WriteEventsCond(t, df.path, metricPrefixIs(df.metricPrefix))
		})
	}
}
