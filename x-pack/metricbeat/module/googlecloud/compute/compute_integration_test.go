// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build googlecloud

package compute

import (
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/googlecloud/metrics"

	"github.com/elastic/beats/v7/libbeat/common"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
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
		{"googlecloud.compute.instance", "./_meta/data.json"},
		{"googlecloud.compute.instance.disk", "./_meta/data_disk.json"},
		{"googlecloud.compute.instance.network", "./_meta/data_network.json"},
		{"googlecloud.compute.instance.cpu", "./_meta/data_cpu.json"},
		{"googlecloud.compute.firewall", "./_meta/data_firewall.json"},
		{"googlecloud.compute.instance.memory", "./_meta/data_memory.json"},
	}

	config := metrics.GetConfigForTest(t, "compute")

	for _, df := range dataFiles {
		metricSet := mbtest.NewFetcher(t, config)
		t.Run(fmt.Sprintf("metric prefix: %s", df.metricPrefix), func(t *testing.T) {
			metricSet.WriteEventsCond(t, df.path, metricPrefixIs(df.metricPrefix))
		})
	}
}
