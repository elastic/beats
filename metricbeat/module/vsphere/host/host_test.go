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
	"github.com/vmware/govmomi/simulator"
)

func TestFetchEventContents(t *testing.T) {
	model := simulator.ESX()
	if err := model.Create(); err != nil {
		t.Fatal(err)
	}

	ts := model.Service.NewServer()
	defer ts.Close()

	f := mbtest.NewReportingMetricSetV2WithContext(t, getConfig(ts))
	events, errs := mbtest.ReportingFetchV2WithContext(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}

	assert.NotEmpty(t, events)

	event := events[0].MetricSetFields

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	assert.EqualValues(t, "localhost.localdomain", event["name"])

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

	diskCapacity := disk["capacity"].(mapstr.M)
	diskCapacityUsage := diskCapacity["usage"].(mapstr.M)
	assert.GreaterOrEqual(t, diskCapacityUsage["bytes"], int64(0))

	diskDevielatency := disk["devicelatency"].(mapstr.M)
	diskDevielatencyAverage := diskDevielatency["average"].(mapstr.M)
	assert.GreaterOrEqual(t, diskDevielatencyAverage["ms"], int64(0))

	diskLatency := disk["latency"].(mapstr.M)
	diskLatencyTotal := diskLatency["total"].(mapstr.M)
	assert.GreaterOrEqual(t, diskLatencyTotal["ms"], int64(0))

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

	network := event["network"].(mapstr.M)

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

	networkErrors := networkPackets["errors"].(mapstr.M)
	networkErrorsTransmitted := networkErrors["transmitted"].(mapstr.M)
	networkErrorsReceived := networkErrors["received"].(mapstr.M)
	networkErrorsTotal := networkErrors["total"].(mapstr.M)
	assert.GreaterOrEqual(t, networkErrorsTransmitted["count"], int64(0))
	assert.GreaterOrEqual(t, networkErrorsReceived["count"], int64(0))
	assert.GreaterOrEqual(t, networkErrorsTotal["count"], int64(0))

	networkMulticast := networkPackets["multicast"].(mapstr.M)
	networkMulticastTransmitted := networkMulticast["transmitted"].(mapstr.M)
	networkMulticastReceived := networkMulticast["received"].(mapstr.M)
	networkMulticastTotal := networkMulticast["total"].(mapstr.M)
	assert.GreaterOrEqual(t, networkMulticastTransmitted["count"], int64(0))
	assert.GreaterOrEqual(t, networkMulticastReceived["count"], int64(0))
	assert.GreaterOrEqual(t, networkMulticastTotal["count"], int64(0))

	networkDropped := networkPackets["dropped"].(mapstr.M)
	networkDroppedTransmitted := networkDropped["transmitted"].(mapstr.M)
	networkDroppedReceived := networkDropped["received"].(mapstr.M)
	networkDroppedTotal := networkDropped["total"].(mapstr.M)
	assert.GreaterOrEqual(t, networkDroppedTransmitted["count"], int64(0))
	assert.GreaterOrEqual(t, networkDroppedReceived["count"], int64(0))
	assert.GreaterOrEqual(t, networkDroppedTotal["count"], int64(0))
}

func TestData(t *testing.T) {
	model := simulator.ESX()
	if err := model.Create(); err != nil {
		t.Fatal(err)
	}

	ts := model.Service.NewServer()
	defer ts.Close()

	f := mbtest.NewReportingMetricSetV2WithContext(t, getConfig(ts))

	if err := mbtest.WriteEventsReporterV2WithContext(f, t, ""); err != nil {
		t.Fatal("write", err)
	}
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
