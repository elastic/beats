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

package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/simulator"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestFetchEventContents(t *testing.T) {
	model := simulator.VPX()
	err := model.Create()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { model.Remove() })

	ts := model.Service.NewServer()
	t.Cleanup(func() { ts.Close() })

	f := mbtest.NewReportingMetricSetV2WithContext(t, getConfig(ts))
	events, errs := mbtest.ReportingFetchV2WithContext(f)
	require.Empty(t, errs, "expected no error")

	require.NotEmpty(t, events, "didn't get any event, should have gotten at least X")

	event := events[0].MetricSetFields

	t.Logf("Fetched event from %s/%s event: %+v", f.Module().Name(), f.Name(), event)

	assert.NotEmpty(t, event["name"])
	assert.EqualValues(t, true, event["accessible"])
	assert.EqualValues(t, "green", event["status"])

	assert.EqualValues(t, "Network", event["type"])

	config := event["config"].(mapstr.M)
	assert.NotNil(t, config)

	if host, ok := event["host"].(mapstr.M); ok {
		assert.GreaterOrEqual(t, host["count"], 0)
		assert.NotEmpty(t, host["names"])
	}

	if vm, ok := event["vm"].(mapstr.M); ok {
		assert.GreaterOrEqual(t, vm["count"], 0)
		assert.NotEmpty(t, vm["names"])
	}
}

func TestNetworkMetricSetData(t *testing.T) {
	model := simulator.ESX()
	err := model.Create()
	require.NoError(t, err, "failed to create model")
	t.Cleanup(func() { model.Remove() })

	ts := model.Service.NewServer()
	t.Cleanup(func() { ts.Close() })

	f := mbtest.NewReportingMetricSetV2WithContext(t, getConfig(ts))

	err = mbtest.WriteEventsReporterV2WithContext(f, t, "")
	assert.NoError(t, err, "failed to write events with reporter")
}

func getConfig(ts *simulator.Server) map[string]interface{} {
	urlSimulator := ts.URL.Scheme + "://" + ts.URL.Host + ts.URL.Path

	return map[string]interface{}{
		"module":     "vsphere",
		"metricsets": []string{"network"},
		"hosts":      []string{urlSimulator},
		"username":   "user",
		"password":   "pass",
		"insecure":   true,
	}
}
