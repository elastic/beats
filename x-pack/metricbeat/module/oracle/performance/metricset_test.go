// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build integration && oracle
// +build integration,oracle

package performance

import (
	"testing"

	_ "github.com/godror/godror"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/oracle"
)

func TestData(t *testing.T) {
	r := compose.EnsureUp(t, "oracle")

	f := mbtest.NewReportingMetricSetV2WithContext(t, getConfig(r.Host()))

	findKey := func(key string) func(common.MapStr) bool {
		return func(in common.MapStr) bool {
			_, err := in.GetValue("oracle.performance." + key)
			return err == nil
		}
	}

	dataFiles := []struct {
		keyToFind string
		filePath  string
	}{
		{
			keyToFind: "buffer_pool",
			filePath:  "./_meta/cache_data.json",
		},
		{
			keyToFind: "username",
			filePath:  "./_meta/cursor_by_username_and_machine_data.json",
		},
		{
			keyToFind: "lock_requests",
			filePath:  "",
		},
	}

	for _, dataFile := range dataFiles {
		t.Run(dataFile.filePath, func(t *testing.T) {
			if err := mbtest.WriteEventsReporterV2WithContextCond(f, t, dataFile.filePath, findKey(dataFile.keyToFind)); err != nil {
				t.Fatal("write", err)
			}
		})
	}
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "oracle",
		"metricsets": []string{"performance"},
		"hosts":      []string{oracle.GetOracleConnectionDetails(host)},
		"username":   "sys",
		"password":   "Oradoc_db1",
	}
}
