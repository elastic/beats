// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build gcp

package loadbalancing

import (
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp/metrics"
)

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
