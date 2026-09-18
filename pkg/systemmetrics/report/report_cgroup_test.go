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

//go:build linux

package report

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/systemmetrics/metric/system/cgroup"
	"github.com/elastic/beats/v7/pkg/systemmetrics/metric/system/resolve"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// The cgroup readers store throttling time in microseconds on both hierarchies,
// while the self-monitoring metric is nanoseconds, so the reporters have to
// scale. These tests pin that scaling; without it v2 under-reports by 1000x.
func TestReportMetricsCgroupThrottledNS(t *testing.T) {
	const testPID = 4242

	tests := map[string]struct {
		hostfs func(t *testing.T) string
		report func(*testing.T, *cgroup.Reader, monitoring.Visitor)
		// The counter in cpu.stat, in the unit the kernel uses for that
		// hierarchy: nanoseconds on v1, microseconds on v2.
		expectedNS int64
	}{
		"v1 converts the nanosecond counter back to nanoseconds": {
			hostfs: func(t *testing.T) string { return writeV1Hostfs(t, testPID, 352597023453) },
			report: func(t *testing.T, r *cgroup.Reader, v monitoring.Visitor) {
				ReportMetricsCGV1(logptest.NewTestingLogger(t, ""), testPID, r, v)
			},
			expectedNS: 352597023000, // truncated to whole microseconds on the way through
		},
		"v2 scales the microsecond counter to nanoseconds": {
			hostfs: func(t *testing.T) string { return writeV2Hostfs(t, testPID, 11931900) },
			report: func(t *testing.T, r *cgroup.Reader, v monitoring.Visitor) {
				ReportMetricsCGV2(logptest.NewTestingLogger(t, ""), testPID, r, v)
			},
			expectedNS: 11931900000,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			reader, err := cgroup.NewReaderOptions(cgroup.ReaderOptions{
				RootfsMountpoint: resolve.NewTestResolver(tc.hostfs(t)),
				Logger:           logptest.NewTestingLogger(t, ""),
			})
			require.NoError(t, err, "could not build a cgroup reader for the test hostfs")

			reg := monitoring.NewRegistry()
			monitoring.NewFunc(reg, "cgroup", func(_ monitoring.Mode, v monitoring.Visitor) {
				v.OnRegistryStart()
				defer v.OnRegistryFinished()
				tc.report(t, reader, v)
			}, monitoring.Report)

			collected := map[string]any{}
			reg.Do(monitoring.Full, func(key string, val any) { collected[key] = val })

			assert.Equal(t, tc.expectedNS, collected["cgroup.cpu.stats.throttled.ns"],
				"throttled time must be reported in nanoseconds, got metrics: %v", collected)
			assert.Equal(t, int64(7), collected["cgroup.cpu.stats.throttled.periods"],
				"throttled periods must be passed through unscaled, got metrics: %v", collected)
		})
	}
}

// writeV1Hostfs lays out a minimal cgroups v1 hierarchy that the reader accepts
// as a hostfs, with throttledTime nanoseconds of CPU throttling for pid.
func writeV1Hostfs(t *testing.T, pid int, throttledTime uint64) string {
	t.Helper()
	root := t.TempDir()
	cpuDir := filepath.Join(root, "sys/fs/cgroup/cpu/testgroup")

	writeFiles(t, map[string]string{
		"proc/cgroups": "#subsys_name\thierarchy\tnum_cgroups\tenabled\ncpu\t2\t1\t1\n",
		// Mountpoints have to sit under the hostfs root, that is how
		// SubsystemMountpoints() tells our mounts apart from the host's.
		"proc/self/mountinfo": fmt.Sprintf(
			"30 23 0:26 / %s rw,relatime shared:9 - cgroup cgroup rw,cpu\n",
			filepath.Join(root, "sys/fs/cgroup/cpu")),
		fmt.Sprintf("proc/%d/cgroup", pid): "2:cpu:/testgroup\n",

		"sys/fs/cgroup/cpu/testgroup/cpu.stat": fmt.Sprintf(
			"nr_periods 100\nnr_throttled 7\nthrottled_time %d\n", throttledTime),
		"sys/fs/cgroup/cpu/testgroup/cpu.cfs_period_us": "100000\n",
		"sys/fs/cgroup/cpu/testgroup/cpu.cfs_quota_us":  "-1\n",
		"sys/fs/cgroup/cpu/testgroup/cpu.shares":        "1024\n",
		"sys/fs/cgroup/cpu/testgroup/cpu.rt_period_us":  "1000000\n",
		"sys/fs/cgroup/cpu/testgroup/cpu.rt_runtime_us": "0\n",
	}, root)
	require.DirExists(t, cpuDir, "the v1 cpu controller directory must exist")

	return root
}

// writeV2Hostfs lays out a minimal cgroups v2 hierarchy that the reader accepts
// as a hostfs, with throttledUsec microseconds of CPU throttling for pid.
func writeV2Hostfs(t *testing.T, pid int, throttledUsec uint64) string {
	t.Helper()
	root := t.TempDir()

	writeFiles(t, map[string]string{
		"proc/cgroups": "#subsys_name\thierarchy\tnum_cgroups\tenabled\ncpu\t0\t1\t1\n",
		"proc/self/mountinfo": fmt.Sprintf(
			"30 23 0:26 / %s rw,relatime shared:9 - cgroup2 cgroup2 rw\n",
			filepath.Join(root, "sys/fs/cgroup")),
		fmt.Sprintf("proc/%d/cgroup", pid): "0::/testgroup\n",

		"sys/fs/cgroup/testgroup/cpu.stat": fmt.Sprintf(
			"usage_usec 500\nuser_usec 300\nsystem_usec 200\nnr_periods 100\nnr_throttled 7\nthrottled_usec %d\n",
			throttledUsec),
		"sys/fs/cgroup/testgroup/cpu.max":    "max 100000\n",
		"sys/fs/cgroup/testgroup/cpu.weight": "100\n",
	}, root)

	return root
}

func writeFiles(t *testing.T, files map[string]string, root string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755), "could not create %s", filepath.Dir(path))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600), "could not write %s", path)
	}
}
