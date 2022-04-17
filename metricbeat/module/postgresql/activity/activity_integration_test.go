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

package activity

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/tests/compose"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
	"github.com/menderesk/beats/v7/metricbeat/module/postgresql"
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

	assert.Contains(t, event, "pid")
	assert.True(t, event["pid"].(int64) > 0)

	// Check event fields
	if _, isQuery := event["database"]; isQuery {
		db_oid := event["database"].(common.MapStr)["oid"].(int64)
		assert.True(t, db_oid > 0)

		assert.Contains(t, event, "user")
		assert.Contains(t, event["user"].(common.MapStr), "name")
		assert.Contains(t, event["user"].(common.MapStr), "id")
	} else {
		assert.Contains(t, event, "backend_type")
		assert.Contains(t, event, "wait_event")
		assert.Contains(t, event, "wait_event_type")
	}
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "postgresql")

	f := mbtest.NewFetcher(t, getConfig(service.Host()))

	dbNameKey := "postgresql.activity.database.name"
	f.WriteEventsCond(t, "", func(event common.MapStr) bool {
		_, err := event.GetValue(dbNameKey)
		return err == nil
	})
	f.WriteEventsCond(t, "./_meta/data_backend.json", func(event common.MapStr) bool {
		_, err := event.GetValue(dbNameKey)
		return err != nil
	})
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "postgresql",
		"metricsets": []string{"activity"},
		"hosts":      []string{postgresql.GetDSN(host)},
		"username":   postgresql.GetEnvUsername(),
		"password":   postgresql.GetEnvPassword(),
	}
}
