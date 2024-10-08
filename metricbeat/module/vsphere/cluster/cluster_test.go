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

package cluster

import (
	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/simulator"
)

func TestFetchEventContents(t *testing.T) {
	// Creating a new simulator model with VPX server to collect broad range of data.
	model := simulator.VPX()
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

	testEvent := mapstr.M{
		"name": "DC0_C0",
		"host": mapstr.M{
			"count": 3,
			"names": []string{"DC0_C0_H0", "DC0_C0_H1", "DC0_C0_H2"},
		},
		"datastore": mapstr.M{
			"count": 1,
			"names": []string{"LocalDS_0"},
		},
		"network": mapstr.M{
			"count": 3,
			"names": []string{"VM Network", "DVS0-DVUplinks-9", "DC0_DVPG0"},
		},
	}

	assert.Exactly(t, event, testEvent)
}

func TestClusterMetricSetData(t *testing.T) {
	model := simulator.VPX()
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
		"metricsets": []string{"cluster"},
		"hosts":      []string{urlSimulator},
		"username":   "user",
		"password":   "pass",
		"insecure":   true,
	}
}
