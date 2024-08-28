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

package host

import (
	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/simulator"
)

func TestFetchEventContents(t *testing.T) {
	model := simulator.ESX()
	err := model.Create()
	require.NoError(t, err, "failed to create model")
	t.Cleanup(func() { model.Remove() })

	ts := model.Service.NewServer()
	t.Cleanup(func() { ts.Close() })

	f := mbtest.NewReportingMetricSetV2WithContext(t, getConfig(ts))
	events, errs := mbtest.ReportingFetchV2WithContext(f)
	require.Empty(t, errs, "expected no error")

	require.NotEmpty(t, events, "didn't get any event, should have gotten at least X")

	event := events[0].MetricSetFields

	t.Logf("Fetched event from %s/%s event: %+v", f.Module().Name(), f.Name(), event)

	assert.EqualValues(t, "localhost.localdomain", event["name"])

	vm := event["vm"].(mapstr.M)
	assert.NotNil(t, vm["names"])
	assert.GreaterOrEqual(t, vm["count"], 0)

	dataStore := event["datastore"].(mapstr.M)
	assert.NotNil(t, dataStore["names"])
	assert.GreaterOrEqual(t, dataStore["count"], 0)

	assert.NotNil(t, event["network_names"])
	network := event["network"].(mapstr.M)
	assert.NotNil(t, network["names"])
	assert.GreaterOrEqual(t, network["count"], 0)

	assert.NotNil(t, event["status"])

	assert.GreaterOrEqual(t, event["uptime"], int32(0))

	cpu := event["cpu"].(mapstr.M)

	cpuUsed := cpu["used"].(mapstr.M)
	assert.EqualValues(t, 67, cpuUsed["mhz"])

	cpuTotal := cpu["total"].(mapstr.M)
	assert.EqualValues(t, 4588, cpuTotal["mhz"])

	cpuFree := cpu["free"].(mapstr.M)
	assert.EqualValues(t, 4521, cpuFree["mhz"])

	disk := event["disk"].(mapstr.M)

	diskCapacity, ok := disk["capacity"].(mapstr.M)
	if ok {
		diskCapacityUsage := diskCapacity["usage"].(mapstr.M)
		assert.GreaterOrEqual(t, diskCapacityUsage["bytes"], int64(0))
	}

	diskDevielatency, ok := disk["devicelatency"].(mapstr.M)
	if ok {
		diskDevielatencyAverage := diskDevielatency["average"].(mapstr.M)
		assert.GreaterOrEqual(t, diskDevielatencyAverage["ms"], int64(0))
	}
	diskLatency, ok := disk["latency"].(mapstr.M)
	if ok {
		diskLatencyTotal := diskLatency["total"].(mapstr.M)
		assert.GreaterOrEqual(t, diskLatencyTotal["ms"], int64(0))
	}

	diskTotal := disk["total"].(mapstr.M)
	assert.GreaterOrEqual(t, diskTotal["bytes"], int64(0))

	diskRead := disk["read"].(mapstr.M)
	assert.GreaterOrEqual(t, diskRead["bytes"], int64(0))

	diskWrite := disk["write"].(mapstr.M)
	assert.GreaterOrEqual(t, diskWrite["bytes"], int64(0))

	memory := event["memory"].(mapstr.M)

	memoryUsed := memory["used"].(mapstr.M)
	assert.EqualValues(t, uint64(1472200704), memoryUsed["bytes"])

	memoryTotal := memory["total"].(mapstr.M)
	assert.EqualValues(t, uint64(4294430720), memoryTotal["bytes"])

	memoryFree := memory["free"].(mapstr.M)
	assert.EqualValues(t, uint64(2822230016), memoryFree["bytes"])

	network = event["network"].(mapstr.M)

	networkBandwidth := network["bandwidth"].(mapstr.M)
	networkBandwidthTransmitted := networkBandwidth["transmitted"].(mapstr.M)
	networkBandwidthReceived := networkBandwidth["received"].(mapstr.M)
	networkBandwidthTotal := networkBandwidth["total"].(mapstr.M)
	assert.GreaterOrEqual(t, networkBandwidthTransmitted["bytes"], int64(0))
	assert.GreaterOrEqual(t, networkBandwidthReceived["bytes"], int64(0))
	assert.GreaterOrEqual(t, networkBandwidthTotal["bytes"], int64(0))

	networkPackets := network["packets"].(mapstr.M)
	networkPacketsTransmitted := networkPackets["transmitted"].(mapstr.M)
	networkPacketsReceived := networkPackets["received"].(mapstr.M)
	assert.GreaterOrEqual(t, networkPacketsTransmitted["count"], int64(0))
	assert.GreaterOrEqual(t, networkPacketsReceived["count"], int64(0))

	networkErrors, ok := networkPackets["errors"].(mapstr.M)
	if ok {
		networkErrorsTransmitted := networkErrors["transmitted"].(mapstr.M)
		networkErrorsReceived := networkErrors["received"].(mapstr.M)
		networkErrorsTotal := networkErrors["total"].(mapstr.M)
		assert.GreaterOrEqual(t, networkErrorsTransmitted["count"], int64(0))
		assert.GreaterOrEqual(t, networkErrorsReceived["count"], int64(0))
		assert.GreaterOrEqual(t, networkErrorsTotal["count"], int64(0))
	}

	networkMulticast := networkPackets["multicast"].(mapstr.M)
	if networkMulticastTransmitted, ok := networkMulticast["transmitted"].(mapstr.M); ok {
		assert.GreaterOrEqual(t, networkMulticastTransmitted["count"], int64(0))
	}
	if networkMulticastReceived, ok := networkMulticast["received"].(mapstr.M); ok {
		assert.GreaterOrEqual(t, networkMulticastReceived["count"], int64(0))
	}
	if networkMulticastTotal, ok := networkMulticast["total"].(mapstr.M); ok {
		assert.GreaterOrEqual(t, networkMulticastTotal["count"], int64(0))
	}

	if networkDropped, ok := networkPackets["dropped"].(mapstr.M); ok {
		if networkDroppedTransmitted, ok := networkDropped["transmitted"].(mapstr.M); ok {
			assert.GreaterOrEqual(t, networkDroppedTransmitted["count"], int64(0))
		}
		if networkDroppedReceived, ok := networkDropped["received"].(mapstr.M); ok {
			assert.GreaterOrEqual(t, networkDroppedReceived["count"], int64(0))
		}
		if networkDroppedTotal, ok := networkDropped["total"].(mapstr.M); ok {
			assert.GreaterOrEqual(t, networkDroppedTotal["count"], int64(0))
		}
	}
}

func TestHostMetricSetData(t *testing.T) {
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
		"metricsets": []string{"host"},
		"hosts":      []string{urlSimulator},
		"username":   "user",
		"password":   "pass",
		"insecure":   true,
	}
}
