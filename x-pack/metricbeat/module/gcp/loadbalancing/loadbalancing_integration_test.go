// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && gcp
// +build integration,gcp

package loadbalancing

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/common"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
	"github.com/menderesk/beats/v7/x-pack/metricbeat/module/gcp/metrics"
)

func TestFetch(t *testing.T) {
	config := metrics.GetConfigForTest(t, "loadbalancing")
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
	metricPrefixIs := func(metricPrefix string) func(e common.MapStr) bool {
		return func(e common.MapStr) bool {
			v, err := e.GetValue(metricPrefix)
			return err == nil && v != nil
		}
	}

	dataFiles := []struct {
		metricPrefix string
		path         string
	}{
		{"gcp.loadbalancing", "./_meta/data.json"},
		{"gcp.loadbalancing.https", "./_meta/data_https.json"},
		{"gcp.loadbalancing.l3", "./_meta/data_l3.json"},
		{"gcp.loadbalancing.tcp_ssl_proxy", "./_meta/data_tcp_ssl_proxy.json"},
	}

	config := metrics.GetConfigForTest(t, "loadbalancing")

	for _, df := range dataFiles {
		metricSet := mbtest.NewFetcher(t, config)
		t.Run(fmt.Sprintf("metric prefix: %s", df.metricPrefix), func(t *testing.T) {
			metricSet.WriteEventsCond(t, df.path, metricPrefixIs(df.metricPrefix))
		})
	}
}
