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

package status

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestFetchTCP(t *testing.T) {
	service := compose.EnsureUp(t, "uwsgi_tcp")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig("tcp", service.Host()))
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)

	t.Log(events)
	totals := findItems(events, "total")
	assert.Equal(t, 1, len(totals))
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "uwsgi_http")

	f := mbtest.NewFetcher(t, getConfig("http", service.Host()))
	f.WriteEventsCond(t, "", func(event common.MapStr) bool {
		isOverall, _ := event.HasKey("uwsgi.status.total")
		return isOverall
	})
	f.WriteEventsCond(t, "_meta/data_core.json", func(event common.MapStr) bool {
		isCore, _ := event.HasKey("uwsgi.status.core")
		return isCore
	})
	f.WriteEventsCond(t, "_meta/data_worker.json", func(event common.MapStr) bool {
		isWorker, _ := event.HasKey("uwsgi.status.worker")
		return isWorker
	})
}

func TestFetchHTTP(t *testing.T) {
	service := compose.EnsureUp(t, "uwsgi_http")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig("http", service.Host()))
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)

	t.Log(events)
	totals := findItems(events, "total")
	assert.Equal(t, 1, len(totals))
}

func getConfig(scheme string, host string) map[string]interface{} {
	conf := map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
	}

	switch scheme {
	case "http", "https":
		conf["hosts"] = []string{"http://" + host}
	default:
		conf["hosts"] = []string{"tcp://" + host}
	}
	return conf
}
