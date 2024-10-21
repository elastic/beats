// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/elastic/elastic-agent-libs/api/npipe"
	"github.com/elastic/elastic-agent-libs/kibana"
	"github.com/elastic/elastic-agent/internal/pkg/agent/application/paths"
	"github.com/elastic/elastic-agent/pkg/control/v2/cproto"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools"
	"github.com/elastic/elastic-agent/pkg/testing/tools/estools"
	"github.com/elastic/elastic-agent/pkg/utils"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

// ExtendedRunner is the main test runner
type ExtendedRunner struct {
	suite.Suite
	info                   *define.Info
	agentFixture           *atesting.Fixture
	ESHost                 string
	healthCheckTime        time.Duration
	healthCheckRefreshTime time.Duration

	resourceWatchers []StatusWatcher
}

// BeatStats is used to parse the result of a /stats call to the beat control socket
type BeatStats struct {
	Beat struct {
		Runtime struct {
			Goroutines int `json:"goroutines"`
		} `json:"runtime"`
	} `json:"beat"`
}

func TestLongRunningAgentForLeaks(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Group: "fleet",
		Stack: &define.Stack{},
		Local: false, // requires Agent installation
		Sudo:  true,  // requires Agent installation
		OS: []define.OS{
			{Type: define.Linux},
			{Type: define.Windows},
		},
	})

	if os.Getenv("TEST_LONG_RUNNING") == "" {
		t.Skip("not running extended test unless TEST_LONG_RUNNING is set")
	}

	suite.Run(t, &ExtendedRunner{info: info,
		healthCheckTime:        time.Minute * 6,
		healthCheckRefreshTime: time.Second * 20,
		resourceWatchers: []StatusWatcher{ // select which tests to run
			&handleMonitor{},
			&goroutinesMonitor{},
		}})
}

func (runner *ExtendedRunner) SetupSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "install", "-v", "github.com/mingrammer/flog@latest")
	out, err := cmd.CombinedOutput()
	require.NoError(runner.T(), err, "got out: %s", string(out))

	cmd = exec.CommandContext(ctx, "flog", "-t", "log", "-f", "apache_error", "-o", "/var/log/httpd/error_log", "-b", "50485760", "-p", "1048576")
	out, err = cmd.CombinedOutput()
	require.NoError(runner.T(), err, "got out: %s", string(out))

	policyUUID := uuid.Must(uuid.NewV4()).String()
	installOpts := atesting.InstallOpts{
		NonInteractive: true,
		Force:          true,
		Privileged:     true,
	}

	fixture, err := define.NewFixtureFromLocalBuild(runner.T(), define.Version())
	require.NoError(runner.T(), err)
	runner.agentFixture = fixture

	basePolicy := kibana.AgentPolicy{
		Name:        "test-policy-" + policyUUID,
		Namespace:   "default",
		Description: "Test policy " + policyUUID,
		MonitoringEnabled: []kibana.MonitoringEnabledOption{
			kibana.MonitoringEnabledLogs,
			kibana.MonitoringEnabledMetrics,
		},
	}

	policyResp, err := tools.InstallAgentWithPolicy(ctx, runner.T(), installOpts, runner.agentFixture, runner.info.KibanaClient, basePolicy)
	require.NoError(runner.T(), err)

	_, err = tools.InstallPackageFromDefaultFile(ctx, runner.info.KibanaClient, "system", "1.53.1", "agent_long_test_base_system_integ.json", uuid.Must(uuid.NewV4()).String(), policyResp.ID)
	require.NoError(runner.T(), err)

	_, err = tools.InstallPackageFromDefaultFile(ctx, runner.info.KibanaClient, "apache", "1.17.0", "agent_long_test_apache.json", uuid.Must(uuid.NewV4()).String(), policyResp.ID)
	require.NoError(runner.T(), err)

}

func (runner *ExtendedRunner) TestHandleLeak() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	testRuntime := os.Getenv("LONG_TEST_RUNTIME")
	if testRuntime == "" {
		testRuntime = "15m"
	}

	// block until we're sure agent is healthy
	runner.CheckHealthAtStartup(ctx)

	// initialize the resource watchers that will sit and look for usage patterns
	for _, mon := range runner.resourceWatchers {
		mon.Init(ctx, runner.T(), runner.agentFixture)
	}

	testDuration, err := time.ParseDuration(testRuntime)
	require.NoError(runner.T(), err)

	timer := time.NewTimer(testDuration)
	defer timer.Stop()

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	done := false
	for !done {
		select {
		case <-timer.C:
			done = true
		case <-ticker.C:
			err := runner.agentFixture.IsHealthy(ctx)
			require.NoError(runner.T(), err)
			// iterate through our watchers, update them
			for _, mon := range runner.resourceWatchers {
				mon.Update(runner.T(), runner.agentFixture)
			}
		}
	}

	// we're measuring the handle usage as y=mx+b
	// if the slope is increasing above a certain rate, fail the test
	// A number of factors can change the slope during a test; shortened runtime (lots of handles allocated in the first few seconds, producing an upward slope),
	// filebeat trying to open a large number of log files, etc
	// handleSlopeFailure := 0.1
	for _, mon := range runner.resourceWatchers {
		handleSlopeFailure := 0.1

		for _, handle := range mon.GetSlopeHandlers() {
			err := handle.Run()
			require.NoError(runner.T(), err)
			runner.T().Logf("=============================== %s", handle.Name())
			handleSlope := handle.GetSlope()
			require.LessOrEqual(runner.T(), handleSlope, handleSlopeFailure, "increase in count of gorutines exceeded threshold for %s: %s", handle.Name(), handle.Debug())
			runner.T().Logf("Passed check for %s; component: %s", mon.Name(), handle.Name())
			runner.T().Logf("===============================")
		}
	}

	status, err := runner.agentFixture.ExecStatus(ctx)
	require.NoError(runner.T(), err)

	// post-test: make sure that we actually ingested logs.
	docs, err := estools.GetResultsForAgentAndDatastream(ctx, runner.info.ESClient, "apache.error", status.Info.ID)
	assert.NoError(runner.T(), err, "error fetching apache logs")
	assert.Greater(runner.T(), docs.Hits.Total.Value, 0, "could not find any matching apache logs for agent ID %s", status.Info.ID)
	runner.T().Logf("Generated %d apache logs", docs.Hits.Total.Value)

	docs, err = estools.GetResultsForAgentAndDatastream(ctx, runner.info.ESClient, "system.cpu", status.Info.ID)
	assert.NoError(runner.T(), err, "error fetching system metrics")
	assert.Greater(runner.T(), docs.Hits.Total.Value, 0, "could not find any matching system metrics for agent ID %s", status.Info.ID)
	runner.T().Logf("Generated %d system events", docs.Hits.Total.Value)
}

// CheckHealthAtStartup ensures all the beats and agent are healthy and working before we continue
func (runner *ExtendedRunner) CheckHealthAtStartup(ctx context.Context) {
	// because we need to separately fetch the PIDs, wait until everything is healthy before we look for running beats
	compDebugName := ""
	require.Eventually(runner.T(), func() bool {
		allHealthy := true
		status, err := runner.agentFixture.ExecStatus(ctx)
		if err != nil {
			runner.T().Logf("agent status returned an error: %v", err)
			return false
		}

		apacheMatch := "logfile-apache"
		foundApache := false
		systemMatch := "system/metrics"
		foundSystem := false

		for _, comp := range status.Components {
			// make sure the components include the expected integrations
			for _, v := range comp.Units {
				runner.T().Logf("unit ID: %s", v.UnitID)
				// the full unit ID will be something like "log-default-logfile-cef-3f0764f0-4ade-4f46-9ead-f2f0f7865676"
				if !foundApache && strings.Contains(v.UnitID, apacheMatch) {
					foundApache = true
				}
				if !foundSystem && strings.Contains(v.UnitID, systemMatch) {
					foundSystem = true
				}
				runner.T().Logf("unit state: %s", v.Message)
				if v.State != int(cproto.State_HEALTHY) {
					allHealthy = false
				}
			}
			runner.T().Logf("component state: %s", comp.Message)
			if comp.State != int(cproto.State_HEALTHY) {
				compDebugName = comp.Name
				allHealthy = false
			}
		}
		return allHealthy && foundApache && foundSystem
	}, runner.healthCheckTime, runner.healthCheckRefreshTime, "install never became healthy: components did not return a healthy state: %s", compDebugName)
}

/*
=============================================================================
Watchers for checking resource usage
=============================================================================
*/

type StatusWatcher interface {
	Init(ctx context.Context, t *testing.T, status *atesting.Fixture)
	Update(t *testing.T, fixture *atesting.Fixture)
	GetSlopeHandlers() []tools.Slope
	Name() string
}

// goroutineWatcher tracks individual components under test
type goroutineWatcher struct {
	httpClient    http.Client
	regGoroutines tools.Slope
	compName      string
}

// goroutinesMonitor tracks thread usage across agent
type goroutinesMonitor struct {
	handles   []goroutineWatcher
	startTime time.Time
}

func (gm *goroutinesMonitor) Init(ctx context.Context, t *testing.T, fixture *atesting.Fixture) {
	oldTop := paths.Top()
	paths.SetTop("/opt/Elastic/Agent")
	// fetch the unit ID of the component, use that to generate the path to the unix socket
	status, err := fixture.ExecStatus(ctx)
	if err != nil {
		t.Logf("agent status returned an error: %v", err)
	}

	for _, comp := range status.Components {
		unitId := comp.ID
		socketPath := utils.SocketURLWithFallback(unitId, paths.TempDir())
		handlesReg := tools.NewSlope(comp.Name)
		watcher := goroutineWatcher{
			regGoroutines: handlesReg,
			compName:      comp.Name,
			httpClient: http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
						if runtime.GOOS != "windows" {
							path := strings.Replace(socketPath, "unix://", "", -1)
							return net.Dial("unix", path)
						} else {
							if strings.HasPrefix(socketPath, "npipe:///") {
								path := strings.TrimPrefix(socketPath, "npipe:///")
								socketPath = `\\.\pipe\` + path
							}
							return npipe.DialContext(socketPath)(ctx, "", "")
						}

					},
				},
			},
		}
		gm.handles = append(gm.handles, watcher)

	}
	gm.startTime = time.Now()
	paths.SetTop(oldTop)
}

func (gm *goroutinesMonitor) Update(t *testing.T, fixture *atesting.Fixture) {
	// reach out to the unix sockets to get the raw stats that includes a count of gorutines
	for _, comp := range gm.handles {
		resp, err := comp.httpClient.Get("http://unix/stats")
		require.NoError(t, err)
		respRaw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		data := BeatStats{}
		err = json.Unmarshal(respRaw, &data)
		require.NoError(t, err)
		resp.Body.Close()

		comp.regGoroutines.AddDatapoint(float64(data.Beat.Runtime.Goroutines), time.Since(gm.startTime))
	}
}

func (gm *goroutinesMonitor) GetSlopeHandlers() []tools.Slope {
	// handleSlopeFailure := 0.1
	slopes := []tools.Slope{}
	for _, handle := range gm.handles {
		slopes = append(slopes, handle.regGoroutines)
	}
	return slopes
}

func (gm *goroutinesMonitor) Name() string {
	return "goroutines"
}

// process watcher is used to track the handle counts of individual running beats.
type processWatcher struct {
	handle     types.Process
	pid        int
	name       string
	regHandles tools.Slope
}

// handleMonitor tracks the rate of increase (slope) in running file handles
type handleMonitor struct {
	handles   []processWatcher
	startTime time.Time
}

func (handleMon *handleMonitor) Init(ctx context.Context, t *testing.T, fixture *atesting.Fixture) {
	// track running beats
	// the `last 30s` metrics tend to report gauges, which we can't use for calculating a derivative.
	// so separately fetch the PIDs
	pidInStatusMessageRegex := regexp.MustCompile(`[\d]+`)
	status, err := fixture.ExecStatus(ctx)
	if err != nil {
		t.Logf("agent status returned an error: %v", err)
	}

	for _, comp := range status.Components {
		pidStr := pidInStatusMessageRegex.FindString(comp.Message)
		pid, err := strconv.ParseInt(pidStr, 10, 64)
		require.NoError(t, err)

		handle, err := sysinfo.Process(int(pid))
		require.NoError(t, err)
		handlesReg := tools.NewSlope(comp.Name)

		t.Logf("created handle watcher for %s (%d)", comp.Name, pid)
		handleMon.handles = append(handleMon.handles, processWatcher{handle: handle, pid: int(pid), name: comp.Name, regHandles: handlesReg})
	}
	handleMon.startTime = time.Now()
}

func (handleMon *handleMonitor) Update(t *testing.T, _ *atesting.Fixture) {
	// for each running process, collect memory and handles
	for _, handle := range handleMon.handles {
		ohc, ok := handle.handle.(types.OpenHandleCounter)
		if ok {
			handleCount, err := ohc.OpenHandleCount()
			require.NoError(t, err)
			handle.regHandles.AddDatapoint(float64(handleCount), time.Since(handleMon.startTime))
		}
	}
}

func (handleMon *handleMonitor) GetSlopeHandlers() []tools.Slope {
	slopes := []tools.Slope{}
	for _, handle := range handleMon.handles {
		slopes = append(slopes, handle.regHandles)
	}
	return slopes
}

func (gm *handleMonitor) Name() string {
	return "handles"
}
