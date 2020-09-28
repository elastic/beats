// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build googlecloud

package storage

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
		{"googlecloud.storage", "./_meta/data.json"},
		{"googlecloud.storage.authz", "./_meta/data_authz.json"},
		{"googlecloud.storage.network", "./_meta/data_network.json"},
		{"googlecloud.storage.storage", "./_meta/data_storage.json"},
	}

	config := metrics.GetConfigForTest(t, "storage")

	for _, df := range dataFiles {
		metricSet := mbtest.NewFetcher(t, config)
		t.Run(fmt.Sprintf("metric prefix: %s", df.metricPrefix), func(t *testing.T) {
			metricSet.WriteEventsCond(t, df.path, metricPrefixIs(df.metricPrefix))
		})
	}
}
