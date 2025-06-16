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

package systemtests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

// DockerTestRunner is a simple test framework for running a given go test inside a container.
// In order for this framework to work fully, tests running under this framework must use
// systemtests.DockerTestResolver() to fetch the hostfs, and also set debug-level logging via `logp`.
type DockerTestRunner struct {
	Runner *testing.T
	// Privileged is equivalent to the `--privileged` flag passed to `docker run`. Sets elevated permissions.
	Privileged bool
	// Sets the filepath passed to `go test`
	Basepath string
	// Testname will run a given test if set
	Testname string
	// Container name passed to `docker run`.
	Container string
	// RunAsUser will run the container as a non-root user if set to the given username
	// equivalent to `--user=USER`
	RunAsUser string
	// CgroupNSMode sets the cgroup namespace for the container. Newer versions of docker
	// will default to a private namespace. Unexpected namespace values have resulted in bugs.
	CgroupNSMode container.CgroupnsMode
	// Verbose enables debug-level logging
	Verbose bool
	// FatalLogMessages  will fail the test if a given string appears in the log output for the test.
	// Useful for turning non-fatal errors into fatal errors.
	// These are just passed to strings.Contains(). I.e. []string{"Non-fatal error"}
	FatalLogMessages []string
	// MonitorPID will tell tests to specifically check the correctness of process-level monitoring for this PID
	MonitorPID int
	// CreateHostProcess: this will start a process with the following args outside of the container,
	// and use the integration tests to monitor it.
	// Useful as "monitor random running processes" as a test heuristic tends to be flaky.
	// This will overrite MonitorPID, so only set either this or MonitorPID.
	CreateHostProcess *exec.Cmd
}

// RunResult returns the logs and return code from the container
type RunResult struct {
	ReturnCode int64
	Stderr     string
	Stdout     string
}

type testCase struct {
	nsmode container.CgroupnsMode
	priv   bool
	user   string
}

func (tc testCase) String() string {
	return fmt.Sprintf("nsmode:%s_priv:%v_user:%s", tc.nsmode, tc.priv, tc.user)
}

// CreateAndRunPermissionMatrix is a helper that runs RunTestsOnDocker() across a range of possible docker settings.
// If a given array value is supplied in the method arguments,
// it will be used to override the value in DockerTestRunner, and run the test for as many times as there are supplied values.
// For example, if privilegedValues=[true, false], then RunTestsOnDocker() will be run twice,
// setting the privileged flag differently for each run.
// if an array argument is empty, the default value set in the DockerTestRunner instance will be used.
func (tr *DockerTestRunner) CreateAndRunPermissionMatrix(ctx context.Context,
	cgroupNSValues []container.CgroupnsMode, privilegedValues []bool, runAsUserValues []string) {

	cases := []testCase{}

	if len(cgroupNSValues) == 0 {
		cgroupNSValues = []container.CgroupnsMode{tr.CgroupNSMode}
	}

	if len(privilegedValues) == 0 {
		privilegedValues = []bool{tr.Privileged}
	}

	if len(runAsUserValues) == 0 {
		runAsUserValues = []string{tr.RunAsUser}
	}

	// Create a test matrix of every possible case.
	// This might seem like overkill, but cgroup settings and docker permissions values produce some exciting edge cases. Just run all of them.
	for _, ns := range cgroupNSValues {
		for _, user := range runAsUserValues {
			for _, privSetting := range privilegedValues {
				cases = append(cases, testCase{nsmode: ns, priv: privSetting, user: user})
			}
		}
	}

	tr.Runner.Logf("Running %d tests", len(cases))

	baseRunner := tr.Runner // some odd recursion happens here if we just refer to tr.Runner
	for _, tc := range cases {
		baseRunner.Run(tc.String(), func(t *testing.T) {
			runner := tr
			runner.Runner = t
			runner.CgroupNSMode = tc.nsmode
			runner.Privileged = tc.priv
			runner.RunAsUser = tc.user
			runner.RunTestsOnDocker(ctx)
		})
	}

}

// RunTestsOnDocker runs a provided test, or all the package tests
// (as in `go test ./...`) inside a docker container with the host's root FS mounted as /hostfs.
// This framework relies on the tests using DockerTestResolver().
// If docker returns !0 or if there's a matching string entry from FatalLogMessages in stdout/stderr,
// this will fail the test
func (tr *DockerTestRunner) RunTestsOnDocker(ctx context.Context) {
	// do we want to run on windows? Much of what we're testing, such as host
	// cgroup monitoring, is invalid.
	if runtime.GOOS != "linux" {
		tr.Runner.Skip("Tests only supported on Linux.")
	}

	log := logp.L()
	if tr.Basepath == "" {
		tr.Basepath = "./..."
	}

	if tr.Container == "" {
		tr.Container = "golang:latest"
	}

	// setup and run

	apiClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	require.NoError(tr.Runner, err)
	defer apiClient.Close()

	_, err = apiClient.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		tr.Runner.Skipf("got error in container list, docker isn't installed or not running: %s", err)
	}

	// create monitored process, if we need to
	tr.createMonitoredProcess(ctx)

	resp := tr.createTestContainer(ctx, apiClient)

	log.Infof("running test...")
	result := tr.runContainerTest(ctx, apiClient, resp)

	// check for failures

	require.Equal(tr.Runner, int64(0), result.ReturnCode, "got bad docker return code. stdout: %s \nstderr: %s", result.Stdout, result.Stderr)

	if tr.Verbose {
		fmt.Fprintf(os.Stdout, "stderr: %s\n", result.Stderr)
		fmt.Fprintf(os.Stdout, "stdout: %s\n", result.Stdout)
	}

	// iterate by lines to make this easier to read
	if len(tr.FatalLogMessages) > 0 {
		for _, badLine := range tr.FatalLogMessages {
			for _, line := range strings.Split(result.Stdout, "\n") {
				require.NotContains(tr.Runner, line, badLine)
			}
			for _, line := range strings.Split(result.Stderr, "\n") {
				// filter our the go mod package download messages
				if !strings.Contains(line, "go: downloading") {
					require.NotContains(tr.Runner, line, badLine)
				}

			}
		}

	}

}

// createTestContainer creates a container with the given test path and test name
func (tr *DockerTestRunner) createTestContainer(ctx context.Context, apiClient *client.Client) container.CreateResponse {
	reader, err := apiClient.ImagePull(ctx, tr.Container, image.PullOptions{})
	require.NoError(tr.Runner, err, "error pulling image")
	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)
	require.NoError(tr.Runner, err, "error copying image")

	wdCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	wdPath, err := wdCmd.CombinedOutput()
	require.NoError(tr.Runner, err, "error finding root path")

	cwd := strings.TrimSpace(string(wdPath))
	logp.L().Infof("using cwd: %s", cwd)

	testRunCmd := []string{"go", "test", "-v", tr.Basepath}
	if tr.Testname != "" {
		testRunCmd = append(testRunCmd, "-run", tr.Testname)
	}

	mountPath := "/hostfs"

	containerEnv := []string{fmt.Sprintf("HOSTFS=%s", mountPath)}
	// used by a few vendored libaries
	containerEnv = append(containerEnv, "HOST_PROC=%s", mountPath)
	if tr.Privileged {
		containerEnv = append(containerEnv, "PRIVILEGED=1")
	}

	if tr.MonitorPID != 0 {
		containerEnv = append(containerEnv, fmt.Sprintf("MONITOR_PID=%d", tr.MonitorPID))
	}

	gomodcacheCmd := exec.Command("go", "env", "GOMODCACHE")
	gomodcacheValue, err := gomodcacheCmd.CombinedOutput()
	require.NoError(tr.Runner, err)
	gomodcacheValue = bytes.TrimSuffix(gomodcacheValue, []byte("\n"))
	require.NotEmpty(tr.Runner, gomodcacheValue)

	resp, err := apiClient.ContainerCreate(ctx, &container.Config{
		Image:      tr.Container,
		Cmd:        testRunCmd,
		Tty:        false,
		WorkingDir: "/app",
		Env:        containerEnv,
		User:       tr.RunAsUser,
	}, &container.HostConfig{
		CgroupnsMode: tr.CgroupNSMode,
		Privileged:   tr.Privileged,
		Binds:        []string{fmt.Sprintf("/:%s", mountPath), fmt.Sprintf("%s:/app", cwd)},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: string(gomodcacheValue),
				Target: "/go/pkg/mod",
			},
		},
	}, nil, nil, "")
	require.NoError(tr.Runner, err, "error creating container")

	return resp
}

func (tr *DockerTestRunner) runContainerTest(ctx context.Context, apiClient *client.Client, resp container.CreateResponse) RunResult {
	err := apiClient.ContainerStart(ctx, resp.ID, container.StartOptions{})
	require.NoError(tr.Runner, err, "error starting container")

	res := RunResult{}

	statusCh, errCh := apiClient.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		require.NoError(tr.Runner, err, "error in container")
	case status := <-statusCh:
		res.ReturnCode = status.StatusCode
	}

	out, err := apiClient.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	require.NoError(tr.Runner, err, "error fetching logs")

	stdout := bytes.NewBufferString("")
	stderr := bytes.NewBufferString("")
	_, err = stdcopy.StdCopy(stdout, stderr, out)
	require.NoError(tr.Runner, err, "error copying logs")
	res.Stderr = stderr.String()
	res.Stdout = stdout.String()

	return res
}

func (tr *DockerTestRunner) createMonitoredProcess(ctx context.Context) {
	log := logp.L()
	// if user has specified a process to monitor, start it now
	// skip if the process has already been created
	if tr.CreateHostProcess != nil && tr.CreateHostProcess.Process == nil {
		// We don't need to do this in a channel, but it prevents races between this goroutine
		// and the rest of test framework
		startPid := make(chan int)
		log.Infof("Creating test Process...")
		go func() {
			err := tr.CreateHostProcess.Start()
			// if the process fails to start up, the resulting tests will fail anyway, so just log it
			assert.NoError(tr.Runner, err, "error starting monitor process")
			startPid <- tr.CreateHostProcess.Process.Pid

		}()
		select {
		case pid := <-startPid:
			tr.MonitorPID = pid
		case <-ctx.Done():
		}
		log.Infof("Monitoring pid %d", tr.MonitorPID)
	}
}
