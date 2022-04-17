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

package jmx

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/tests/compose"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "jolokia")

	for _, config := range getConfigs(service.Host()) {
		f := mbtest.NewReportingMetricSetV2Error(t, config)
		events, errs := mbtest.ReportingFetchV2Error(f)
		assert.Empty(t, errs)
		assert.NotEmpty(t, events)
		t.Logf("%s/%s events: %+v", f.Module().Name(), f.Name(), events)
	}
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "jolokia")

	for _, config := range getConfigs(service.Host()) {
		f := mbtest.NewReportingMetricSetV2Error(t, config)
		events, errs := mbtest.ReportingFetchV2Error(f)
		assert.Empty(t, errs)
		assert.NotEmpty(t, events)

		if err := mbtest.WriteEventsReporterV2Error(f, t, ""); err != nil {
			t.Fatal("write", err)
		}
	}
}

func getConfigs(host string) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"module":     "jolokia",
			"metricsets": []string{"jmx"},
			"hosts":      []string{host},
			"namespace":  "testnamespace",
			"jmx.mappings": []map[string]interface{}{
				{
					"mbean": "java.lang:type=Runtime",
					"attributes": []map[string]string{
						{
							"attr":  "Uptime",
							"field": "uptime",
						},
					},
				},
				{
					"mbean": "java.lang:type=GarbageCollector,name=ConcurrentMarkSweep",
					"attributes": []map[string]string{
						{
							"attr":  "CollectionTime",
							"field": "gc.cms_collection_time",
						},
						{
							"attr":  "CollectionCount",
							"field": "gc.cms_collection_count",
						},
					},
				},
				{
					"mbean": "java.lang:type=Memory",
					"attributes": []map[string]string{
						{
							"attr":  "HeapMemoryUsage",
							"field": "memory.heap_usage",
						},
						{
							"attr":  "NonHeapMemoryUsage",
							"field": "memory.non_heap_usage",
						},
					},
				},
			},
		},
		{
			"module":     "jolokia",
			"metricsets": []string{"jmx"},
			"hosts":      []string{host},
			"namespace":  "testnamespace",
			"jmx.mappings": []map[string]interface{}{
				{
					"mbean": "Catalina:name=*,type=ThreadPool",
					"attributes": []map[string]string{
						{
							"attr":  "maxConnections",
							"field": "max_connections",
						},
						{
							"attr":  "port",
							"field": "port",
						},
					},
				},
				{
					"mbean": "Catalina:type=Server",
					"attributes": []map[string]string{
						{
							"attr":  "serverNumber",
							"field": "server_number_dosntconnect",
						},
					},
					"target": &TargetBlock{
						URL:      "service:jmx:rmi:///jndi/rmi://localhost:7091/jmxrmi",
						User:     "monitorRole",
						Password: "IGNORE",
					},
				},
				{
					"mbean": "Catalina:type=Server",
					"attributes": []map[string]string{
						{
							"attr":  "serverInfo",
							"field": "server_info_proxy",
						},
					},
					"target": &TargetBlock{
						URL:      "service:jmx:rmi:///jndi/rmi://localhost:7091/jmxrmi",
						User:     "monitorRole",
						Password: "QED",
					},
				},
			},
		},
		{
			"module":      "jolokia",
			"metricsets":  []string{"jmx"},
			"hosts":       []string{host},
			"namespace":   "testnamespace",
			"http_method": "GET",
			"jmx.mappings": []map[string]interface{}{
				{
					"mbean": "java.lang:type=GarbageCollector,name=ConcurrentMarkSweep",
					"attributes": []map[string]string{
						{
							"attr":  "CollectionTime",
							"field": "gc.cms_collection_time",
						},
						{
							"attr":  "CollectionCount",
							"field": "gc.cms_collection_count",
						},
					},
				},
				{
					"mbean": "java.lang:type=Memory",
					"attributes": []map[string]string{
						{
							"attr":  "HeapMemoryUsage",
							"field": "memory.heap_usage",
						},
						{
							"attr":  "NonHeapMemoryUsage",
							"field": "memory.non_heap_usage",
						},
					},
				},
				{
					"mbean": "java.lang:type=Runtime",
					"attributes": []map[string]string{
						{
							"attr":  "Uptime",
							"field": "uptime",
						},
					},
				},
			},
		},
	}
}
