// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build integration
// +build integration

package database

import (
	"testing"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"
	"github.com/elastic/beats/v8/metricbeat/module/postgresql"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "postgresql")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	event := events[0].MetricSetFields

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check event fields
	db_oid := event["oid"].(int64)
	assert.True(t, db_oid >= 0)
	assert.Contains(t, event, "name")
	_, ok := event["name"].(string)
	assert.True(t, ok)

	rows := event["rows"].(common.MapStr)
	assert.Contains(t, rows, "returned")
	assert.Contains(t, rows, "fetched")
	assert.Contains(t, rows, "inserted")
	assert.Contains(t, rows, "updated")
	assert.Contains(t, rows, "deleted")
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "postgresql")

	getOid := func(event common.MapStr) int {
		oid, err := event.GetValue("postgresql.database.oid")
		require.NoError(t, err)

		switch oid := oid.(type) {
		case int:
			return oid
		case int64:
			return int(oid)
		}
		t.Log(event)
		t.Fatalf("no numeric oid in event: %v (%T)", oid, oid)
		return 0
	}

	f := mbtest.NewFetcher(t, getConfig(service.Host()))
	f.WriteEventsCond(t, "", func(event common.MapStr) bool {
		return getOid(event) != 0
	})
	f.WriteEventsCond(t, "./_meta/data_shared.json", func(event common.MapStr) bool {
		return getOid(event) == 0
	})
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "postgresql",
		"metricsets": []string{"database"},
		"hosts":      []string{postgresql.GetDSN(host)},
		"username":   postgresql.GetEnvUsername(),
		"password":   postgresql.GetEnvPassword(),
	}
}
