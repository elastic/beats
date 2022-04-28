// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && gcp
// +build integration,gcp

package compute

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp/metrics"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestFetch(t *testing.T) {
	config := metrics.GetConfigForTest(t, "compute")
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
		{"gcp.compute.instance", "./_meta/data.json"},
		{"gcp.compute.instance.disk", "./_meta/data_disk.json"},
		{"gcp.compute.instance.network", "./_meta/data_network.json"},
		{"gcp.compute.instance.cpu", "./_meta/data_cpu.json"},
		{"gcp.compute.firewall", "./_meta/data_firewall.json"},
		{"gcp.compute.instance.memory", "./_meta/data_memory.json"},
	}

	config := metrics.GetConfigForTest(t, "compute")

	for _, df := range dataFiles {
		metricSet := mbtest.NewFetcher(t, config)
		t.Run(fmt.Sprintf("metric prefix: %s", df.metricPrefix), func(t *testing.T) {
			metricSet.WriteEventsCond(t, df.path, metricPrefixIs(df.metricPrefix))
		})
	}
}
