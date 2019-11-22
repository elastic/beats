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

package consumer

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	// Register input module and metricset
	_ "github.com/elastic/beats/metricbeat/module/jolokia"
	_ "github.com/elastic/beats/metricbeat/module/jolokia/jmx"
)

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "kafka",
		compose.UpWithTimeout(600*time.Second),
		compose.UpWithAdvertisedHostEnvFileForPort(9092),
	)

	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.HostForPort(8774)))
	err := mbtest.WriteEventsReporterV2Error(ms, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "kafka",
		compose.UpWithTimeout(600*time.Second),
		compose.UpWithAdvertisedHostEnvFileForPort(9092),
	)
	reporter := &mbtest.CapturingReporterV2{}

	metricSet := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.HostForPort(8774)))
	metricSet.Fetch(reporter)

	e := mbtest.StandardizeEvent(metricSet, reporter.GetEvents()[0])
	t.Logf("%s/%s event: %+v", metricSet.Module().Name(), metricSet.Name(), e.Fields.StringToPrint())
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "kafka",
		"metricsets": []string{"consumer"},
		"hosts":      []string{host},
	}
}
