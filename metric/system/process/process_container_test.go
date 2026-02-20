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

package process

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-system-metrics/dev-tools/systemtests"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup"
)

// ======================================== NOTE:
// The tests here are meant to be run from the containerized framework in ./tests
// However, they are designed so that `go test` can run them normally as well

func TestContainerMonitoringFromInsideContainer(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	testStats := Stats{
		CPUTicks:      true,
		EnableCgroups: true,
		EnableNetwork: false,
		Hostfs:        systemtests.DockerTestResolver(logger),
		Procs:         []string{".*"},
		CgroupOpts: cgroup.ReaderOptions{
			RootfsMountpoint: systemtests.DockerTestResolver(logger),
			Logger:           logger,
		},
		Logger: logger,
	}
	err := testStats.Init()
	require.NoError(t, err)

	stats, err := testStats.GetSelf()
	require.NoError(t, err)
	if runtime.GOOS == "linux" {
		if stats.Cgroup == nil {
			t.Skip("https://github.com/elastic/elastic-agent-system-metrics/issues/270")
		}
		cgstats, err := stats.Cgroup.Format()
		require.NoError(t, err)
		require.NotEmpty(t, cgstats)
	}

	require.NotEmpty(t, stats.Cmdline)
	require.NotEmpty(t, stats.Username)
	require.NotZero(t, stats.Pid)
}

func TestSelfMonitoringFromInsideContainer(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	testStats := Stats{
		CPUTicks:      true,
		EnableCgroups: true,
		EnableNetwork: false,
		Procs:         []string{".*"},
		CgroupOpts: cgroup.ReaderOptions{
			Logger: logger,
		},
		Logger: logger,
	}
	err := testStats.Init()
	require.NoError(t, err)

	stats, err := testStats.GetSelf()
	require.NoError(t, err)
	if runtime.GOOS == "linux" {
		if stats.Cgroup == nil {
			t.Skip("https://github.com/elastic/elastic-agent-system-metrics/issues/270")
		}
		cgstats, err := stats.Cgroup.Format()
		require.NoError(t, err)
		require.NotEmpty(t, cgstats)
	}

	require.NotEmpty(t, stats.Cmdline)
	require.NotEmpty(t, stats.Username)
	require.NotZero(t, stats.Pid)
}

func TestSystemHostFromContainer(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	testStats := Stats{
		CPUTicks:      true,
		EnableCgroups: true,
		EnableNetwork: false,
		Hostfs:        systemtests.DockerTestResolver(logger),
		Procs:         []string{".*"},
		CgroupOpts: cgroup.ReaderOptions{
			RootfsMountpoint: systemtests.DockerTestResolver(logger),
			Logger:           logger,
		},
		Logger: logger,
	}
	err := testStats.Init()
	require.NoError(t, err)

	// two modes to the test:
	// 1) if the runner specified a PID, use that, check for validity
	// 2) if no PID specified, just fetch all PIDs, let the test use the logs to decide if we failed

	if userPid := os.Getenv("MONITOR_PID"); userPid != "" {
		intPid, err := strconv.ParseInt(userPid, 10, 32)
		require.NoError(t, err, "error parsing MONITOR_PID")
		result, err := testStats.GetOne(int(intPid))
		require.NoError(t, err, "error reading MONITOR_PID")
		validateProcResult(t, result)
	} else {
		_, roots, err := testStats.Get()
		require.True(t, isNonFatal(err), fmt.Sprintf("Fatal error: %s", err))

		for _, proc := range roots {
			t.Logf("proc: %d: %s", proc["process"].(map[string]any)["pid"],
				proc["process"].(map[string]any)["command_line"])
		}
	}
}

// validate test results.
// these are largely heuristic-based, and will check for
// failures related to past bugs, common issues, etc, etc
func validateProcResult(t *testing.T, result mapstr.M) {
	_, privilegedMode := os.LookupEnv("PRIVILEGED")
	cgroupNSMode := os.Getenv("CGROUPNSMODE")
	userID := os.Getuid()
	formatArgs := []any{
		"privileged=%t userID=%d cgroupNSMode=%s result=%s ",
		privilegedMode, userID, cgroupNSMode, result.String(),
	}

	usr, err := user.Current()
	require.NoError(t, err, formatArgs...)

	gotUser, _ := result["username"].(string)

	gotPpid, ok := result["ppid"].(int)
	assert.True(t, ok, formatArgs...)

	// if we're root or the same user as the pid, check `exe`
	// kernel procs also don't have `exe`
	if (privilegedMode && (userID == 0 || usr.Name == gotUser)) && gotPpid != 2 {
		assert.Contains(t, result, "exe", formatArgs...)
	}

	// if privileged or root, look for data from /proc/[pid]/io
	if privilegedMode && userID == 0 {
		ioReadBytes := result["io"].(map[string]any)["read_char"]
		assert.NotNil(t, ioReadBytes, formatArgs...)
	}

	// check thread count
	assert.Contains(t, result, "num_threads", formatArgs...)

	if runtime.GOOS == "linux" {
		// Cgroups may not be available when:
		// - Running as non-root user (permission denied accessing cgroup files)
		// - Private PID namespace with unresolvable cgroup paths (e.g., /../..)
		// These are treated as non-fatal errors in the metrics collection code.
		// TODO: fix this
		// See: https://github.com/elastic/elastic-agent-system-metrics/issues/270
		if cgroupNSMode == "host" && userID == 0 {
			assert.Contains(t, result, "cgroup", formatArgs...)
		} else {
			t.Log("WARN: skipping 'cgroup' check, this is because of known issue https://github.com/elastic/elastic-agent-system-metrics/issues/270")
			t.Logf(formatArgs[0].(string), formatArgs[1:]...)
		}
	}
}
