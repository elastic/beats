// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux darwin

package operation

import (
	"context"
	"testing"
	"time"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/configrequest"
	operatorCfg "github.com/elastic/beats/v7/x-pack/agent/pkg/agent/operation/config"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/stateresolver"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/app/monitoring"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/app/monitoring/beats"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/process"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/retry"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/state"
)

func TestGenerateSteps(t *testing.T) {
	const sampleOutput = "sample-output"
	operator, _ := getMonitorableTestOperator(t, "tests/scripts")

	type testCase struct {
		Name           string
		Config         *monitoring.Config
		ExpectedSteps  int
		FilebeatStep   bool
		MetricbeatStep bool
	}

	testCases := []testCase{
		testCase{"NO monitoring", &monitoring.Config{MonitorLogs: false, MonitorMetrics: false}, 0, false, false},
		testCase{"FB monitoring", &monitoring.Config{MonitorLogs: true, MonitorMetrics: false}, 1, true, false},
		testCase{"MB monitoring", &monitoring.Config{MonitorLogs: false, MonitorMetrics: true}, 1, false, true},
		testCase{"ALL monitoring", &monitoring.Config{MonitorLogs: true, MonitorMetrics: true}, 2, true, true},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			steps := operator.generateMonitoringSteps(tc.Config, "8.0", sampleOutput)
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

func getMonitorableTestOperator(t *testing.T, installPath string) (*Operator, *operatorCfg.Config) {
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
		MonitoringConfig: &monitoring.Config{
			MonitorMetrics: true,
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
	operator, err := NewOperator(ctx, l, "p1", cfg, fetcher, installer, stateResolver, nil)
	if err != nil {
		t.Fatal(err)
	}

	monitor := beats.NewMonitor("dummmy", "p1234", &artifact.Config{OperatingSystem: "linux", InstallPath: "/install/path"}, true, true)
	operator.apps["dummy"] = &testMonitorableApp{monitor: monitor}

	return operator, operatorConfig
}

type testMonitorableApp struct {
	monitor *beats.Monitor
}

func (*testMonitorableApp) Name() string                                              { return "" }
func (*testMonitorableApp) Start(_ context.Context, cfg map[string]interface{}) error { return nil }
func (*testMonitorableApp) Stop()                                                     {}
func (*testMonitorableApp) Configure(_ context.Context, config map[string]interface{}) error {
	return nil
}
func (*testMonitorableApp) State() state.State            { return state.State{} }
func (a *testMonitorableApp) Monitor() monitoring.Monitor { return a.monitor }
