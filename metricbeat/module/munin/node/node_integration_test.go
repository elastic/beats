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

package node

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestNode(t *testing.T) {
	runner := compose.TestRunner{Service: "munin"}
	runner.Run(t, compose.Suite{
		"Fetch": testFetch,
		"Data":  testData,
	})
}

func testFetch(t *testing.T, r compose.R) {
	f := mbtest.NewReportingMetricSetV2(t, getConfig(r.Host()))
	events, errs := mbtest.ReportingFetchV2(f)

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
		events[0].BeatEvent("munin", "node").Fields.StringToPrint())
}

func testData(t *testing.T, r compose.R) {
	config := getConfig(r.Host())
	f := mbtest.NewReportingMetricSetV2(t, config)
	err := mbtest.WriteEventsReporterV2(f, t, ".")
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "munin",
		"metricsets": []string{"node"},
		"hosts":      []string{host},
	}
}
