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

// +build integration,linux

package cluster_status

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/mb/testing/flags"
)

func TestData(t *testing.T) {
	if !*flags.DataFlag {
		t.Skip("Flaky test: https://github.com/elastic/beats/issues/22612")
	}

	service := compose.EnsureUpWithTimeout(t, 120, "ceph-api")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))

	if err := mbtest.WriteEventsReporterV2Error(f, t, ""); err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "ceph",
		"metricsets": []string{"cluster_status"},
		"hosts":      []string{host},
	}
}
