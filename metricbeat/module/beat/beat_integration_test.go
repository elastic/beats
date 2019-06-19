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

package beat_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/beat"
	_ "github.com/elastic/beats/metricbeat/module/beat/stats"
)

var metricSets = []string{
	"stats",
}

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "metricbeat")

	for _, metricSet := range metricSets {
		f := mbtest.NewReportingMetricSetV2Error(t, beat.GetConfig(metricSet))
		events, errs := mbtest.ReportingFetchV2Error(f)

		assert.Empty(t, errs)
		if !assert.NotEmpty(t, events) {
			t.FailNow()
		}

		t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
			events[0].BeatEvent("beat", metricSet).Fields.StringToPrint())
	}
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "metricbeat")

	for _, metricSet := range metricSets {
		config := getConfig(metricSet)
		f := mbtest.NewReportingMetricSetV2Error(t, config)
		err := mbtest.WriteEventsReporterV2Error(f, t, metricSet)
		if err != nil {
			t.Fatal("write", err)
		}
	}
}

func getConfig(metricset string) map[string]interface{} {
	return map[string]interface{}{
		"module":     beat.ModuleName,
		"metricsets": []string{metricset},
		"hosts":      []string{getEnvHost() + ":" + getEnvPort()},
	}
}

func getEnvHost() string {
	host := os.Getenv("BEAT_HOST")

	if len(host) == 0 {
		host = "localhost"
	}
	return host
}

func getEnvPort() string {
	port := os.Getenv("BEAT_PORT")

	if len(port) == 0 {
		port = "5066"
	}
	return port
}
