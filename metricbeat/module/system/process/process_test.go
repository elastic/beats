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

//go:build darwin || freebsd || linux || windows || aix

package process

import (
	"errors"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	_ "github.com/elastic/beats/v7/metricbeat/module/system"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/process"
)

func TestFetch(t *testing.T) {

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	events, errs := mbtest.ReportingFetchV2Error(f)
	for _, err := range errs {
		assert.ErrorIsf(t, err, process.NonFatalErr{}, "Expected non-fatal error, got %v", err)
	}
	assert.NotEmpty(t, events)

	time.Sleep(2 * time.Second)

	events, errs = mbtest.ReportingFetchV2Error(f)
	for _, err := range errs {
		assert.ErrorIsf(t, err, process.NonFatalErr{}, "Expected non-fatal error, got %v", err)
	}
	assert.NotEmpty(t, events)

	t.Logf("fetched %d events, showing events[0]:", len(events))
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
		events[0].BeatEvent("system", "process").Fields.StringToPrint())
}

func TestFetchDegradeOnPartial(t *testing.T) {
	if runtime.GOOS == "windows" && os.Getenv("CI") == "true" {
		t.Skip("Skip: CI run on windows. It is run as admin, but the test requires to run as non-admin")
	}
	if runtime.GOOS != "windows" && os.Getuid() == 0 {
		t.Skip("Skip: running as root on non-windows, but the test requires to run as non-root")
	}

	config := getConfig()
	config["degrade_on_partial"] = true

	f := mbtest.NewReportingMetricSetV2Error(t, config)

	var errs []error
	_, errs = mbtest.ReportingFetchV2Error(f)
	assert.NotEmpty(t, errs, "expected at least one error, got none")

	for _, err := range errs {
		assert.NotErrorIsf(t, err, &mb.PartialMetricsError{},
			"Expected non-fatal error, got %v", err)
	}
}

func TestFetchSinglePid(t *testing.T) {
	cfg := getConfig()
	cfg["process.pid"] = os.Getpid()

	f := mbtest.NewReportingMetricSetV2Error(t, cfg)
	events, errs := mbtest.ReportingFetchV2Error(f)
	assert.Empty(t, errs)
	require.NotEmpty(t, events)
	assert.Equal(t, os.Getpid(), requireGetSubMap(t, events[0].RootFields, "process")["pid"])
	assert.NotEmpty(t, events[0].MetricSetFields["cpu"])
}

func TestData(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())

	// Do a first fetch to have percentages
	mbtest.ReportingFetchV2Error(f)
	time.Sleep(10 * time.Second)

	err := mbtest.WriteEventsReporterV2Error(f, t, ".")
	if err != nil {
		t.Fatal("write", err)
	}
}

func TestCgroupPressure(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("cgroup stats are only available on Linux")
	}

	cfg := getConfig()
	cfg["process.pid"] = getTestPid(t)

	f := mbtest.NewReportingMetricSetV2Error(t, cfg)
	events, errs := mbtest.ReportingFetchV2Error(f)
	assert.Empty(t, errs)
	require.NotEmpty(t, events)

	// Get cgroup data from the event
	event := events[0]
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
	cgroupData, err := event.MetricSetFields.GetValue("cgroup")
	if errors.Is(err, mapstr.ErrKeyNotFound) {
		t.Skip("cgroup data not available")
	}
	require.NoError(t, err)

	cgroup, ok := cgroupData.(map[string]any)
	require.Truef(t, ok, "unexpected cgroup data type: %T", cgroupData)

	subsystems := []string{"cpu", "memory", "io"}
	if cgroup["path"] == "/" {
		// Sanity check: verify our assertion that the / cgroup result in no subsystem data. If these checks fail, it
		// means a change in expected behavior and this test should be adjusted.
		// Example command to test with / cgroup:
		// sleep 60 &
		// pid=$!
		// sudo sh -c "echo $pid > /sys/fs/cgroup/cgroup.procs"
		// cat /proc/$pid/cgroup
		// MONITOR_PID=$pid go test -run TestCgroupPressure -v ./metricbeat/module/system/process
		for _, subsystem := range subsystems {
			assert.NotContains(t, cgroup, subsystem)
		}
		t.Skip("Process in / cgroup, skipping because of IgnoreRootCgroups")
	}

	for _, subsystem := range subsystems {
		t.Run(subsystem, func(t *testing.T) {
			// Subsystem might not exist depending on cgroup configuration.
			// For example, io accounting is disabled by default in systemd:
			// https://www.freedesktop.org/software/systemd/man/latest/systemd-system.conf.html#DefaultMemoryAccounting=
			if _, ok := cgroup[subsystem]; !ok {
				t.Skipf("%s subsystem not available in this cgroup", subsystem)
			}
			controller := requireGetSubMap(t, cgroup, subsystem)
			t.Run("pressure", func(t *testing.T) {
				// Pressure might not exist, be nil, or be empty depending on kernel configuration
				if _, ok := controller["pressure"]; !ok {
					t.Skip("pressure data not available on this cgroup")
				}
				pressure := requireGetSubMap(t, controller, "pressure")
				if pressure == nil {
					// See https://github.com/elastic/elastic-agent-system-metrics/pull/276
					require.Equal(t, "io", subsystem, "only the io subsystem returns nil when unavailable")
					t.Skip("pressure data not available on this cgroup")
				}
				for _, stall := range []string{"some", "full"} {
					t.Run(stall, func(t *testing.T) {
						checkPressure(t, pressure, stall)
					})
				}
			})
		})
	}
}

func checkPressure(t *testing.T, pressure map[string]any, stall string) {
	// Verify pressure structure has expected fields
	stallMap := requireGetSubMap(t, pressure, stall)

	// Check for time window fields (10, 60, 300 seconds)
	for _, window := range []string{"10", "60", "300"} {
		windowMap := requireGetSubMap(t, stallMap, window)

		// Check for pct field
		assert.Contains(t, windowMap, "pct", "expected pressure.%s.%s.pct to exist", stall, window)
	}

	// Check for total field
	assert.Contains(t, stallMap, "total", "expected pressure.%s.total to exist", stall)
}

func requireGetSubMap(t *testing.T, m map[string]any, key string) map[string]any {
	t.Helper()
	require.Contains(t, m, key)
	rawValue := m[key]
	require.IsType(t, map[string]any{}, rawValue)
	subMap, ok := rawValue.(map[string]any)
	require.True(t, ok)
	return subMap
}

func getTestPid(t *testing.T) int {
	t.Helper()
	if targetPid := os.Getenv("MONITOR_PID"); targetPid != "" {
		intPid, err := strconv.ParseInt(targetPid, 10, 32)
		require.NoError(t, err, "error parsing MONITOR_PID")
		return int(intPid)
	}
	return os.Getpid()
}

func getConfig() map[string]any {
	return map[string]any{
		"module":                        "system",
		"metricsets":                    []string{"process"},
		"processes":                     []string{".*"}, // in case we want a prettier looking example for data.json
		"process.cgroups.enabled":       true,
		"process.include_cpu_ticks":     true,
		"process.cmdline.cache.enabled": true,
		"process.include_top_n":         process.IncludeTopConfig{Enabled: true, ByCPU: 5},
	}
}
