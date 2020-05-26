// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"
	"testing"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configrequest"
	operatorCfg "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/operation/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/stateresolver"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/app"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/app/monitoring"
	monitoringConfig "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/app/monitoring/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/retry"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/state"
)

func TestGenerateSteps(t *testing.T) {
	const sampleOutput = "sample-output"

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
			operator, _ := getMonitorableTestOperator(t, "tests/scripts", m)
			steps := operator.generateMonitoringSteps("8.0", sampleOutput)
			if actualSteps := len(steps); actualSteps != tc.ExpectedSteps {
				t.Fatalf("invalid number of steps, expected %v, got %v", tc.ExpectedSteps, actualSteps)
			}

			var fbFound, mbFound bool
			for _, s := range steps {
				// Filebeat step check
				if s.Process == "filebeat" {
					fbFound = true
					checkStep(t, "filebeat", sampleOutput, s)
				}

				// Metricbeat step check
				if s.Process == "metricbeat" {
					mbFound = true
					checkStep(t, "metricbeat", sampleOutput, s)
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

func checkStep(t *testing.T, stepName string, expectedOutput interface{}, s configrequest.Step) {
	if meta := s.Meta[configrequest.MetaConfigKey]; meta != nil {
		mapstr, ok := meta.(map[string]interface{})
		if !ok {
			t.Fatalf("no meta config for %s step", stepName)
		}

		esOut, ok := mapstr["output"].(map[string]interface{})
		if !ok {
			t.Fatalf("output not found for %s step", stepName)
		}

		if actualOutput := esOut["elasticsearch"]; actualOutput != expectedOutput {
			t.Fatalf("output for %s step does not match. expected: %v, got %v", stepName, expectedOutput, actualOutput)
		}
	}
}

func getMonitorableTestOperator(t *testing.T, installPath string, m monitoring.Monitor) (*Operator, *operatorCfg.Config) {
	operatorConfig := &operatorCfg.Config{
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
	}

	cfg, err := config.NewConfigFrom(operatorConfig)
	if err != nil {
		t.Fatal(err)
	}

	l := getLogger()

	fetcher := &DummyDownloader{}
	installer := &DummyInstaller{}

	stateResolver, err := stateresolver.NewStateResolver(l)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	operator, err := NewOperator(ctx, l, "p1", cfg, fetcher, installer, stateResolver, nil, m)
	if err != nil {
		t.Fatal(err)
	}

	operator.apps["dummy"] = &testMonitorableApp{monitor: m}

	return operator, operatorConfig
}

type testMonitorableApp struct {
	monitor monitoring.Monitor
}

func (*testMonitorableApp) Name() string { return "" }
func (*testMonitorableApp) Start(_ context.Context, _ app.Taggable, cfg map[string]interface{}) error {
	return nil
}
func (*testMonitorableApp) Stop() {}
func (*testMonitorableApp) Configure(_ context.Context, config map[string]interface{}) error {
	return nil
}
func (*testMonitorableApp) State() state.State            { return state.State{} }
func (a *testMonitorableApp) Monitor() monitoring.Monitor { return a.monitor }

type testMonitor struct {
	monitorLogs    bool
	monitorMetrics bool
}

// EnrichArgs enriches arguments provided to application, in order to enable
// monitoring
func (b *testMonitor) EnrichArgs(_ string, _ string, args []string, _ bool) []string { return args }

// Cleanup cleans up all drops.
func (b *testMonitor) Cleanup(string, string) error { return nil }

// Prepare executes steps in order for monitoring to work correctly
func (b *testMonitor) Prepare(string, string, int, int) error { return nil }

// LogPath describes a path where application stores logs. Empty if
// application is not monitorable
func (b *testMonitor) LogPath(string, string) string {
	if !b.monitorLogs {
		return ""
	}
	return "path"
}

// MetricsPath describes a location where application exposes metrics
// collectable by metricbeat.
func (b *testMonitor) MetricsPath(string, string) string {
	if !b.monitorMetrics {
		return ""
	}
	return "path"
}

// MetricsPathPrefixed return metrics path prefixed with http+ prefix.
func (b *testMonitor) MetricsPathPrefixed(string, string) string {
	return "http+path"
}

// Reload reloads state based on configuration.
func (b *testMonitor) Reload(cfg *config.Config) error { return nil }

// IsMonitoringEnabled returns true if monitoring is configured.
func (b *testMonitor) IsMonitoringEnabled() bool { return b.monitorLogs || b.monitorMetrics }

// WatchLogs return true if monitoring is configured and monitoring logs is enabled.
func (b *testMonitor) WatchLogs() bool { return b.monitorLogs }

// WatchMetrics return true if monitoring is configured and monitoring metrics is enabled.
func (b *testMonitor) WatchMetrics() bool { return b.monitorMetrics }
