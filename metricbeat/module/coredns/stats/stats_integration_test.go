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

// +build integration

package stats

import (
	"fmt"
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	compose.EnsureUp(t, "coredns")
	// TODO: Use this function to find if an event is the wanted one
	eventIs := func(eventType string) func(e common.MapStr) bool {
		return func(e common.MapStr) bool {
			hasEvent, _ := e.HasKey(eventType)
			return hasEvent
		}
	}
	dataFiles := []struct {
		eventType string
		path      string
	}{
		{"coredns.stats.panic.count.total", "./_meta/data_panic_event.json"},
		{"coredns.stats.dns.request.count.total", "./_meta/data_request_count_event.json"},
		{"coredns.stats.dns.request.size.bytes", "./_meta/data_size_bytes_event.json"},
		{"coredns.stats.dns.request.duration.ns", "./_meta/data_request_duration_ns_event.json"},
		{"coredns.stats.dns.response.rcode", "./_meta/data_response_rcode_event.json"},
		{"coredns.stats.dns.request.type", "./_meta/data_request_type_event.json"},
	}
	f := mbtest.NewReportingMetricSetV2(t, getConfig())

	for _, df := range dataFiles {
		t.Run(fmt.Sprintf("event type:%s", df.eventType), func(t *testing.T) {
			err := mbtest.WriteEventsReporterV2Cond(f, t, df.path, eventIs(df.eventType))
			if err != nil {
				t.Fatal("write", err)
			}
		})
	}

}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "coredns",
		"metricsets": []string{"stats"},
		"hosts":      []string{GetEnvHost() + ":" + GetEnvPort()},
	}
}

func GetEnvHost() string {
	host := os.Getenv("COREDNS_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

func GetEnvPort() string {
	port := os.Getenv("COREDNS_PORT")

	if len(port) == 0 {
		port = "9153"
	}
	return port
}
