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

package node

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/tests/compose"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
	"github.com/menderesk/beats/v7/metricbeat/module/rabbitmq/mtest"
)

func TestData(t *testing.T) {
	t.Skip("Skipping test as it was not stable. Probably first event is empty.")
	service := compose.EnsureUpWithTimeout(t, 120, "rabbitmq")

	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	err := mbtest.WriteEventsReporterV2Error(ms, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}

func TestFetch(t *testing.T) {
	t.Skip("Skipping test as it was not stable. Probably first event is empty.")
	service := compose.EnsureUpWithTimeout(t, 120, "rabbitmq")

	reporter := &mbtest.CapturingReporterV2{}

	metricSet := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	err := metricSet.Fetch(reporter)
	assert.NoError(t, err)

	e := mbtest.StandardizeEvent(metricSet, reporter.GetEvents()[0])
	t.Logf("%s/%s event: %+v", metricSet.Module().Name(), metricSet.Name(), e.Fields.StringToPrint())
}

func getConfig(host string) map[string]interface{} {
	config := mtest.GetIntegrationConfig(host)
	config["metricsets"] = []string{"node"}
	return config
}
