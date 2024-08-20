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

package resourcepool

import (
	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/simulator"
)

func TestFetchEventContents(t *testing.T) {
	model := simulator.VPX()
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

	assert.EqualValues(t, "Resources", event["name"])
	assert.EqualValues(t, "green", event["status"])

	vm := event["vm"].(mapstr.M)
	assert.NotNil(t, vm["names"])
	assert.GreaterOrEqual(t, vm["count"], 0)

	cpu := event["cpu"].(mapstr.M)

	cpuUsageMhz, ok := cpu["usage.mhz"].(mapstr.M)
	if ok {
		assert.GreaterOrEqual(t, cpuUsageMhz, int64(0))
	}

	cpuUsagPct, ok := cpu["usage.pct"].(mapstr.M)
	if ok {
		assert.GreaterOrEqual(t, cpuUsagPct, int64(0))
	}

	cpuEntitlement, ok := cpu["entitlement"].(mapstr.M)
	if ok {
		assert.GreaterOrEqual(t, cpuEntitlement["mhz"], int64(0))
	}

	cpuActive, ok := cpu["active"].(mapstr.M)
	if ok {
		assert.GreaterOrEqual(t, cpuActive["average.pct"], int64(0))
		assert.GreaterOrEqual(t, cpuActive["max.pct"], int64(0))
	}

	memory := event["memory"].(mapstr.M)

	memoryUsage, ok := memory["usage"].(mapstr.M)
	if ok {
		assert.GreaterOrEqual(t, memoryUsage["pct"], int64(0))
	}

	memoryShared, ok := memory["shared"].(mapstr.M)
	if ok {
		assert.GreaterOrEqual(t, memoryShared["bytes"], int64(0))
	}

	memorySwap, ok := memory["swap"].(mapstr.M)
	if ok {
		assert.GreaterOrEqual(t, memorySwap["bytes"], int64(0))
	}

	memoryEntitlement, ok := memory["entitlement"].(mapstr.M)
	if ok {
		assert.GreaterOrEqual(t, memoryEntitlement["mhz"], int64(0))
	}
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
		"metricsets": []string{"resourcepool"},
		"hosts":      []string{urlSimulator},
		"username":   "user",
		"password":   "pass",
		"insecure":   true,
	}
}
