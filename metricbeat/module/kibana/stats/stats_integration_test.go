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
	"testing"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/kibana/mtest"
)

func TestData(t *testing.T) {
	t.Skip("Skipping until we find a way to conditionally skip this for Kibana < 6.4.0") // FIXME
	compose.EnsureUp(t, "kibana")

	f := mbtest.NewReportingMetricSetV2(t, mtest.GetConfig("stats"))

	// FIXME! See skip above
	// version, err := kibana.GetVersion(f.http, "api/stats")
	// if err != nil {
	// 	t.Fatal("getting kibana version", err)
	// }

	// isStatsAPIAvailable, err := kibana.IsStatsAPIAvailable(version)
	// if err != nil {
	// 	t.Fatal("checking if kibana stats API is available", err)
	// }

	// t.Skip("Kibana stats API is not available until 6.4.0")

	err := mbtest.WriteEventsReporterV2(f, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}
