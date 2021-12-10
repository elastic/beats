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

//go:build darwin || freebsd || linux || windows
// +build darwin freebsd linux windows

package process_summary

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	_ "github.com/elastic/beats/v7/metricbeat/module/system"
)

func TestData(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	err := mbtest.WriteEventsReporterV2Error(f, t, ".")
	if err != nil {
		t.Fatal("write", err)
	}
}

func TestFetch(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	events, errs := mbtest.ReportingFetchV2Error(f)

	require.Empty(t, errs)
	require.NotEmpty(t, events)
	event := events[0].BeatEvent("system", "process_summary").Fields
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
		event.StringToPrint())

	summary, err := event.GetValue("system.process.summary")
	require.NoError(t, err)

	event, ok := summary.(common.MapStr)
	require.True(t, ok)

	if runtime.GOOS == "windows" {
		assert.Contains(t, event, "total")
		assert.Contains(t, event, "sleeping")
		assert.Contains(t, event, "running")
		assert.Contains(t, event, "unknown")
		total := event["sleeping"].(int) + event["running"].(int) + event["unknown"].(int)
		assert.Equal(t, event["total"].(int), total)
	} else {
		assert.Contains(t, event, "total")
		assert.Contains(t, event, "sleeping")
		assert.Contains(t, event, "running")
		assert.Contains(t, event, "idle")
		assert.Contains(t, event, "stopped")
		assert.Contains(t, event, "zombie")
		assert.Contains(t, event, "unknown")
		total := event["sleeping"].(int) + event["running"].(int) + event["idle"].(int) +
			event["stopped"].(int) + event["zombie"].(int) + event["unknown"].(int)

		assert.Equal(t, event["total"].(int), total)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"process_summary"},
	}
}
