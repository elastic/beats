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
	"os"
	"os/user"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-system-metrics/dev-tools/systemtests"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup"
)

// ======================================== NOTE:
// The tests here are meant to be run from the containerized framework in ./tests
// However, they are designed so that `go test` can run them normally as well

// TODO:
// Right now, cgroups checks are disabled because they're just broken for too many edge cases:
// https://github.com/elastic/elastic-agent-system-metrics/issues/139
// https://github.com/elastic/elastic-agent-system-metrics/issues/132
// After those are fixed, this test needs to have cgroups re-enabled

func TestSystemHostFromContainer(t *testing.T) {
	_ = logp.DevelopmentSetup()
	// This is more rigorous version of the unit tests
	// Because this is meant to run in privileged mode,
	// we know that all data should be in the final event, and we should get no errors

	testStats := Stats{CPUTicks: true,
		EnableCgroups: false,
		EnableNetwork: false,
		Hostfs:        systemtests.DockerTestResolver(),
		Procs:         []string{".*"},
		CgroupOpts:    cgroup.ReaderOptions{RootfsMountpoint: systemtests.DockerTestResolver()},
	}
	err := testStats.Init()
	require.NoError(t, err)

	// two modes to the test:
	// 1) if the runner specified a PID, use that, check for validity
	// 2) if no PID specified, just fetch all PIDs, let the test use the logs to decide if we failed

	if userPid, found := os.LookupEnv("MONITOR_PID"); found && userPid != "" {
		intPid, err := strconv.ParseInt(userPid, 10, 32)
		require.NoError(t, err, "error parsing MONITOR_PID")
		result, err := testStats.GetOne(int(intPid))
		require.NoError(t, err, "error reading MONITOR_PID")
		validateProcResult(t, result)
	} else {
		_, roots, err := testStats.Get()
		require.NoError(t, err)

		for _, proc := range roots {
			t.Logf("proc: %d: %s", proc["process"].(map[string]interface{})["pid"],
				proc["process"].(map[string]interface{})["command_line"])
		}
	}
}

// validate test results.
// these are largely heuristic-based, and will check for
// failures related to past bugs, common issues, etc, etc
func validateProcResult(t *testing.T, result mapstr.M) {
	// uncomment if you're trying to figure out what to check
	//t.Logf("got: %s", result.StringToPrint())

	_, privilegedMode := os.LookupEnv("PRIVILEGED")
	user, err := user.Current()
	require.NoError(t, err)
	gotUser, _ := result["username"].(string)

	// if we're root or the same user as the pid, check `exe`
	if privilegedMode && (os.Getuid() == 0 || user.Name == gotUser) {
		exe := result["exe"]
		require.NotNil(t, exe)
	}

	// if privileged or root, look for data from /proc/[pid]/io
	if privilegedMode && os.Getuid() == 0 {
		ioReadBytes := result["io"].(map[string]interface{})["read_char"]
		require.NotNil(t, ioReadBytes)
	}

	// check thread count
	numThreads := result["num_threads"]
	require.NotNil(t, numThreads)

	// see comment at top of file
	// cgroups := result["cgroup"].(map[string]interface{})
	// require.NotNil(t, cgroups)

}
