// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && oracle

package sysmetric

import (
	"testing"

	_ "github.com/godror/godror"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/oracle"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestData(t *testing.T) {
	r := compose.EnsureUp(t, "oracle")

	f := mbtest.NewReportingMetricSetV2WithContext(t, getConfig(r.Host()))

	findKey := func(key string) func(mapstr.M) bool {
		return func(in mapstr.M) bool {
			_, err := in.GetValue("oracle.sysmetric.metrics." + key)
			return err == nil
		}
	}

	dataFiles := []struct {
		keyToFind string
		filePath  string
	}{
		{
			keyToFind: "name",
			filePath:  "./_meta/data.json",
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
		"metricsets": []string{"sysmetric"},
		"hosts":      []string{oracle.GetOracleConnectionDetails(host)},
		"patterns":   []string{"Session%"},
		"username":   "sys",
		"password":   "Oradoc_db1",
	}
}
