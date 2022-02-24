// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.elastic.co/apm"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configrequest"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/stateresolver"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/app"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring"
	monitoringConfig "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/retry"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
)

func TestExportedMetrics(t *testing.T) {
	programName := "testing"
	expectedMetricsName := "metric_name"
	program.SupportedMap[programName] = program.Spec{ExprtedMetrics: []string{expectedMetricsName}}

	exportedMetrics := normalizeHTTPCopyRules(programName)

	exportedMetricFound := false
	for _, kv := range exportedMetrics {
		from, found := kv["from"]
		if !found {
			continue
		}
		to, found := kv["to"]
		if !found {
			continue
		}

		if to != expectedMetricsName {
			continue
		}
		if from != fmt.Sprintf("http.agent.%s", expectedMetricsName) {
			continue
		}
		exportedMetricFound = true
		break
	}

	require.True(t, exportedMetricFound, "exported metric not found")
	delete(program.SupportedMap, programName)
}

func TestGenerateSteps(t *testing.T) {
	const sampleOutput = "sample-output"
	const outputType = "logstash"

	type testCase struct {
		Name           string
		Config         *monitoringConfig.MonitoringConfig
		ExpectedSteps  int
		FilebeatStep   bool
		MetricbeatStep bool
	}

	testCases := []testCase{
		{"NO monitoring", &monitoringConfig.MonitoringConfig{MonitorLogs: false, MonitorMetrics: false}, 0, false, false},
		{"FB monitoring", &monitoringConfig.MonitoringConfig{MonitorLogs: true, MonitorMetrics: false}, 1, true, false},
		{"MB monitoring", &monitoringConfig.MonitoringConfig{MonitorLogs: false, MonitorMetrics: true}, 1, false, true},
		{"ALL monitoring", &monitoringConfig.MonitoringConfig{MonitorLogs: true, MonitorMetrics: true}, 2, true, true},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			m := &testMonitor{monitorLogs: tc.Config.MonitorLogs, monitorMetrics: tc.Config.MonitorMetrics}
			operator := getMonitorableTestOperator(t, "tests/scripts", m, tc.Config)
			steps := operator.generateMonitoringSteps("8.0", outputType, sampleOutput)
			if actualSteps := len(steps); actualSteps != tc.ExpectedSteps {
				t.Fatalf("invalid number of steps, expected %v, got %v", tc.ExpectedSteps, actualSteps)
			}

			var fbFound, mbFound bool
			for _, s := range steps {
				// Filebeat step check
				if s.ProgramSpec.Cmd == "filebeat" {
					fbFound = true
					checkStep(t, "filebeat", outputType, sampleOutput, s)
				}

				// Metricbeat step check
				if s.ProgramSpec.Cmd == "metricbeat" {
					mbFound = true
					checkStep(t, "metricbeat", outputType, sampleOutput, s)
				}
			}

			if tc.FilebeatStep != fbFound {
				t.Fatalf("Steps for filebeat do not match. Was expected: %v, Was found: %v", tc.FilebeatStep, fbFound)
			}

			if tc.MetricbeatStep != mbFound {
				t.Fatalf("Steps for metricbeat do not match. Was expected: %v, Was found: %v", tc.MetricbeatStep, mbFound)
			}
		})
	}
}

func checkStep(t *testing.T, stepName string, outputType string, expectedOutput interface{}, s configrequest.Step) {
	if meta := s.Meta[configrequest.MetaConfigKey]; meta != nil {
		mapstr, ok := meta.(map[string]interface{})
		if !ok {
			t.Fatalf("no meta config for %s step", stepName)
		}

		esOut, ok := mapstr["output"].(map[string]interface{})
		if !ok {
			t.Fatalf("output not found for %s step", stepName)
		}

		if actualOutput := esOut[outputType]; actualOutput != expectedOutput {
			t.Fatalf("output for %s step does not match. expected: %v, got %v", stepName, expectedOutput, actualOutput)
		}
	}
}

func getMonitorableTestOperator(t *testing.T, installPath string, m monitoring.Monitor, mcfg *monitoringConfig.MonitoringConfig) *Operator {
	cfg := &configuration.SettingsConfig{
		RetryConfig: &retry.Config{
			Enabled:      true,
			RetriesCount: 2,
			Delay:        3 * time.Second,
			MaxDelay:     10 * time.Second,
		},
		ProcessConfig: &process.Config{},
		DownloadConfig: &artifact.Config{
			InstallPath:     installPath,
			OperatingSystem: "darwin",
		},
		MonitoringConfig: mcfg,
	}

	l := getLogger()
	agentInfo, _ := info.NewAgentInfo(true)

	fetcher := &DummyDownloader{}
	verifier := &DummyVerifier{}
	installer := &DummyInstallerChecker{}
	uninstaller := &DummyUninstaller{}

	stateResolver, err := stateresolver.NewStateResolver(l)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := server.New(l, "localhost:0", &ApplicationStatusHandler{}, apm.DefaultTracer)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	operator, err := NewOperator(ctx, l, agentInfo, "p1", cfg, fetcher, verifier, installer, uninstaller, stateResolver, srv, nil, m, status.NewController(l))
	if err != nil {
		t.Fatal(err)
	}

	operator.apps["dummy"] = &testMonitorableApp{monitor: m}

	return operator
}

type testMonitorableApp struct {
	monitor monitoring.Monitor
}

func (*testMonitorableApp) Name() string  { return "" }
func (*testMonitorableApp) Started() bool { return false }
func (*testMonitorableApp) Start(_ context.Context, _ app.Taggable, cfg map[string]interface{}) error {
	return nil
}
func (*testMonitorableApp) Stop()     {}
func (*testMonitorableApp) Shutdown() {}
func (*testMonitorableApp) Configure(_ context.Context, config map[string]interface{}) error {
	return nil
}
func (*testMonitorableApp) Spec() program.Spec                                          { return program.Spec{} }
func (*testMonitorableApp) State() state.State                                          { return state.State{} }
func (*testMonitorableApp) SetState(_ state.Status, _ string, _ map[string]interface{}) {}
func (a *testMonitorableApp) Monitor() monitoring.Monitor                               { return a.monitor }
func (a *testMonitorableApp) OnStatusChange(_ *server.ApplicationState, _ proto.StateObserved_Status, _ string, _ map[string]interface{}) {
}

type testMonitor struct {
	monitorLogs    bool
	monitorMetrics bool
}

// EnrichArgs enriches arguments provided to application, in order to enable
// monitoring
func (b *testMonitor) EnrichArgs(_ program.Spec, _ string, args []string, _ bool) []string {
	return args
}

// Cleanup cleans up all drops.
func (b *testMonitor) Cleanup(program.Spec, string) error { return nil }

// Close closes the monitor.
func (b *testMonitor) Close() {}

// Prepare executes steps in order for monitoring to work correctly
func (b *testMonitor) Prepare(program.Spec, string, int, int) error { return nil }

// LogPath describes a path where application stores logs. Empty if
// application is not monitorable
func (b *testMonitor) LogPath(program.Spec, string) string {
	if !b.monitorLogs {
		return ""
	}
	return "path"
}

// MetricsPath describes a location where application exposes metrics
// collectable by metricbeat.
func (b *testMonitor) MetricsPath(program.Spec, string) string {
	if !b.monitorMetrics {
		return ""
	}
	return "path"
}

// MetricsPathPrefixed return metrics path prefixed with http+ prefix.
func (b *testMonitor) MetricsPathPrefixed(program.Spec, string) string {
	return "http+path"
}

// Reload reloads state based on configuration.
func (b *testMonitor) Reload(cfg *config.Config) error { return nil }

// IsMonitoringEnabled returns true if monitoring is configured.
func (b *testMonitor) IsMonitoringEnabled() bool { return b.monitorLogs || b.monitorMetrics }

// MonitoringNamespace returns monitoring namespace configured.
func (b *testMonitor) MonitoringNamespace() string { return "default" }

// WatchLogs return true if monitoring is configured and monitoring logs is enabled.
func (b *testMonitor) WatchLogs() bool { return b.monitorLogs }

// WatchMetrics return true if monitoring is configured and monitoring metrics is enabled.
func (b *testMonitor) WatchMetrics() bool { return b.monitorMetrics }
