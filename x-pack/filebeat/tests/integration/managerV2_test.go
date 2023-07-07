// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

// TestInputReloadUnderElasticAgent will start a Filebeat and cause the input
// reload issue described on https://github.com/elastic/beats/issues/33653.
// In short, a new input for a file needs to be started while there are still
// events from that file in the publishing pipeline, effectively keeping
// the harvester status as `finished: false`, which prevents the new input
// from starting.
//
// This tests ensures Filebeat can gracefully recover from this situation
// and will eventually re-start harvesting the file.
//
// In case of a test failure the directory with Filebeat logs and
// all other supporting files will be kept on build/integration-tests.
//
// Run the tests with -v flag to print the temporary folder used.
func TestInputReloadUnderElasticAgent(t *testing.T) {
	// First things first, ensure ES is running and we can connect to it.
	// If ES is not running, the test will timeout and the only way to know
	// what caused it is going through Filebeat's logs.
	integration.EnsureESIsRunning(t)

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	logFilePath := filepath.Join(filebeat.TempDir(), "flog.log")
	generateLogFile(t, logFilePath)
	var units = [][]*proto.UnitExpected{
		{
			{
				Id:             "output-unit",
				Type:           proto.UnitType_OUTPUT,
				ConfigStateIdx: 1,
				State:          proto.State_HEALTHY,
				LogLevel:       proto.UnitLogLevel_DEBUG,
				Config: &proto.UnitExpectedConfig{
					Id:   "default",
					Type: "elasticsearch",
					Name: "elasticsearch",
					Source: integration.RequireNewStruct(t,
						map[string]interface{}{
							"type":                 "elasticsearch",
							"hosts":                []interface{}{"http://localhost:9200"},
							"username":             "admin",
							"password":             "testing",
							"protocol":             "http",
							"enabled":              true,
							"allow_older_versions": true,
						}),
				},
			},
			{
				Id:             "input-unit-1",
				Type:           proto.UnitType_INPUT,
				ConfigStateIdx: 1,
				State:          proto.State_HEALTHY,
				LogLevel:       proto.UnitLogLevel_DEBUG,
				Config: &proto.UnitExpectedConfig{
					Id:   "log-input",
					Type: "log",
					Name: "log",
					Streams: []*proto.Stream{
						{
							Id: "log-input-1",
							Source: integration.RequireNewStruct(t, map[string]interface{}{
								"enabled": true,
								"type":    "log",
								"paths":   []interface{}{logFilePath},
							}),
						},
					},
				},
			},
		},
		{
			{
				Id:             "output-unit",
				Type:           proto.UnitType_OUTPUT,
				ConfigStateIdx: 1,
				State:          proto.State_HEALTHY,
				LogLevel:       proto.UnitLogLevel_DEBUG,
				Config: &proto.UnitExpectedConfig{
					Id:   "default",
					Type: "elasticsearch",
					Name: "elasticsearch",
					Source: integration.RequireNewStruct(t,
						map[string]interface{}{
							"type":                 "elasticsearch",
							"hosts":                []interface{}{"http://localhost:9200"},
							"username":             "admin",
							"password":             "testing",
							"protocol":             "http",
							"enabled":              true,
							"allow_older_versions": true,
						}),
				},
			},
			{
				Id:             "input-unit-2",
				Type:           proto.UnitType_INPUT,
				ConfigStateIdx: 1,
				State:          proto.State_HEALTHY,
				LogLevel:       proto.UnitLogLevel_DEBUG,
				Config: &proto.UnitExpectedConfig{
					Id:   "log-input",
					Type: "log",
					Name: "log",
					Streams: []*proto.Stream{
						{
							Id: "log-input-2",
							Source: integration.RequireNewStruct(t, map[string]interface{}{
								"enabled": true,
								"type":    "log",
								"paths":   []interface{}{logFilePath},
							}),
						},
					},
				},
			},
		},
	}

	// Once the desired state is reached (aka Filebeat finished applying
	// the policy changes) we still wait for a little bit before sending
	// another policy. This will allow the input to run and get some data
	// into the publishing pipeline.
	//
	// nextState is a helper function that will keep cycling through both
	// elements of the `units` slice. Once one is fully applied, we wait
	// at least 10s then send the next one.
	idx := 0
	waiting := false
	when := time.Now()
	nextState := func() {
		if waiting {
			if time.Now().After(when) {
				idx = (idx + 1) % len(units)
				waiting = false
				return
			}
			return
		}
		waiting = true
		when = time.Now().Add(10 * time.Second)
	}
	server := &mock.StubServerV2{
		// The Beat will call the check-in function multiple times:
		// - At least once at startup
		// - At every state change (starting, configuring, healthy, etc)
		// for every Unit.
		//
		// Because of that we can't rely on the number of times it is called
		// we need some sort of state machine to handle when to send the next
		// policy and when to just re-send the current one.
		//
		// If the Elastic-Agent wants the Beat to keep running the same policy,
		// it will just keep re-sending it every time the Beat calls the check-in
		// method.
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			if management.DoesStateMatch(observed, units[idx], 0) {
				nextState()
			}
			for _, unit := range observed.GetUnits() {
				if state := unit.GetState(); !(state == proto.State_HEALTHY || state != proto.State_CONFIGURING || state == proto.State_STARTING) {
					t.Fatalf("Unit '%s' is not healthy, state: %s", unit.GetId(), unit.GetState().String())
				}
			}
			return &proto.CheckinExpected{
				Units: units[idx],
			}
		},
		ActionImpl: func(response *proto.ActionResponse) error { return nil },
	}

	require.NoError(t, server.Start())
	t.Cleanup(server.Stop)

	filebeat.Start(
		"-E", fmt.Sprintf(`management.insecure_grpc_url_for_testing="localhost:%d"`, server.Port),
		"-E", "management.enabled=true",
	)

	// waitDeadlineOr5Mins looks at the test deadline
	// and returns a reasonable value of waiting for a
	// condition to be met. The possible values are:
	// - if no test deadline is set, return 5 minutes
	// - if a deadline is set and there is less than
	//   0.5 second left, return the time left
	// - otherwise return the time left minus 0.5 second.
	waitDeadlineOr5Min := func() time.Duration {
		deadline, deadileSet := t.Deadline()
		if deadileSet {
			left := deadline.Sub(time.Now())
			final := left - 500*time.Millisecond
			if final <= 0 {
				return left
			}
			return final
		}
		return 5 * time.Minute
	}

	require.Eventually(t, func() bool {
		return filebeat.LogContains("Can only start an input when all related states are finished")
	}, waitDeadlineOr5Min(), 100*time.Millisecond,
		"String 'Can only start an input when all related states are finished' not found on Filebeat logs")

	require.Eventually(t, func() bool {
		return filebeat.LogContains("file 'flog.log' is not finished, will retry starting the input soon")
	}, waitDeadlineOr5Min(), 100*time.Millisecond,
		"String 'file 'flog.log' is not finished, will retry starting the input soon' not found on Filebeat logs")

	require.Eventually(t, func() bool {
		return filebeat.LogContains("ForceReload set to TRUE")
	}, waitDeadlineOr5Min(), 100*time.Millisecond,
		"String 'ForceReload set to TRUE' not found on Filebeat logs")

	require.Eventually(t, func() bool {
		return filebeat.LogContains("Reloading Beats inputs because forceReload is true")
	}, waitDeadlineOr5Min(), 100*time.Millisecond,
		"String 'Reloading Beats inputs because forceReload is true' not found on Filebeat logs")

	require.Eventually(t, func() bool {
		return filebeat.LogContains("ForceReload set to FALSE")
	}, waitDeadlineOr5Min(), 100*time.Millisecond,
		"String 'ForceReload set to FALSE' not found on Filebeat logs")
}

// TestFailedOutputReportsUnhealthy ensures that if an output
// fails to start and returns an error, the manager will set it
// as failed and the inputs will not be started, which means
// staying on the started state.
func TestFailedOutputReportsUnhealthy(t *testing.T) {
	// First things first, ensure ES is running and we can connect to it.
	// If ES is not running, the test will timeout and the only way to know
	// what caused it is going through Filebeat's logs.
	integration.EnsureESIsRunning(t)
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	finalStateReached := false
	var units = []*proto.UnitExpected{
		{
			Id:             "output-unit-borken",
			Type:           proto.UnitType_OUTPUT,
			ConfigStateIdx: 1,
			State:          proto.State_FAILED,
			LogLevel:       proto.UnitLogLevel_DEBUG,
			Config: &proto.UnitExpectedConfig{
				Id:   "default",
				Type: "logstash",
				Name: "logstash",
				Source: integration.RequireNewStruct(t,
					map[string]interface{}{
						"type":    "logstash",
						"invalid": "configuration",
					}),
			},
		},
		// Also add an input unit to make sure it never leaves the
		// starting state
		{
			Id:             "input-unit",
			Type:           proto.UnitType_INPUT,
			ConfigStateIdx: 1,
			State:          proto.State_STARTING,
			LogLevel:       proto.UnitLogLevel_DEBUG,
			Config: &proto.UnitExpectedConfig{
				Id:   "log-input",
				Type: "log",
				Name: "log",
				Streams: []*proto.Stream{
					{
						Id: "log-input",
						Source: integration.RequireNewStruct(t, map[string]interface{}{
							"enabled": true,
							"type":    "log",
							"paths":   "/tmp/foo",
						}),
					},
				},
			},
		},
	}

	server := &mock.StubServerV2{
		// The Beat will call the check-in function multiple times:
		// - At least once at startup
		// - At every state change (starting, configuring, healthy, etc)
		// for every Unit.
		//
		// So we wait until the state matches the desired state
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			if management.DoesStateMatch(observed, units, 0) {
				finalStateReached = true
			}

			return &proto.CheckinExpected{
				Units: units,
			}
		},
		ActionImpl: func(response *proto.ActionResponse) error { return nil },
	}

	require.NoError(t, server.Start())

	filebeat.Start(
		"-E", fmt.Sprintf(`management.insecure_grpc_url_for_testing="localhost:%d"`, server.Port),
		"-E", "management.enabled=true",
	)

	require.Eventually(t, func() bool {
		return finalStateReached
	}, 30*time.Second, 100*time.Millisecond, "Output unit did not report unhealthy")

	t.Cleanup(server.Stop)
}

func TestRecoverFromInvalidOutputConfiguration(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	// Having the log file enables the inputs to start, while it is not
	// strictly necessary for testing output issues, it allows for the
	// input to start which creates a more realistic test case and
	// can help uncover other issues in the startup/shutdown process.
	logFilePath := filepath.Join(filebeat.TempDir(), "flog.log")
	generateLogFile(t, logFilePath)

	logLevel := proto.UnitLogLevel_INFO
	filestreamInputHealthy := proto.UnitExpected{
		Id:             "input-unit-healthy",
		Type:           proto.UnitType_INPUT,
		ConfigStateIdx: 1,
		State:          proto.State_HEALTHY,
		LogLevel:       logLevel,
		Config: &proto.UnitExpectedConfig{
			Id:   "filestream-input",
			Type: "filestream",
			Name: "filestream-input-healty",
			Streams: []*proto.Stream{
				{
					Id: "filestream-input-id",
					Source: integration.RequireNewStruct(t, map[string]interface{}{
						"id":      "filestream-stream-input-id",
						"enabled": true,
						"type":    "filestream",
						"paths":   logFilePath,
					}),
				},
			},
		},
	}

	filestreamInputStarting := proto.UnitExpected{
		Id:             "input-unit-2",
		Type:           proto.UnitType_INPUT,
		ConfigStateIdx: 1,
		State:          proto.State_STARTING,
		LogLevel:       logLevel,
		Config: &proto.UnitExpectedConfig{
			Id:   "filestream-input",
			Type: "filestream",
			Name: "filestream-input-starting",
			Streams: []*proto.Stream{
				{
					Id: "filestream-input-id",
					Source: integration.RequireNewStruct(t, map[string]interface{}{
						"id":      "filestream-stream-input-id",
						"enabled": true,
						"type":    "filestream",
						"paths":   logFilePath,
					}),
				},
			},
		},
	}

	healthyOutput := proto.UnitExpected{
		Id:             "output-unit",
		Type:           proto.UnitType_OUTPUT,
		ConfigStateIdx: 1,
		State:          proto.State_HEALTHY,
		LogLevel:       logLevel,
		Config: &proto.UnitExpectedConfig{
			Id:   "default",
			Type: "elasticsearch",
			Name: "elasticsearch",
			Source: integration.RequireNewStruct(t,
				map[string]interface{}{
					"type":     "elasticsearch",
					"hosts":    []interface{}{"http://localhost:9200"},
					"username": "admin",
					"password": "testing",
					"protocol": "http",
					"enabled":  true,
				}),
		},
	}

	brokenOutput := proto.UnitExpected{
		Id:             "output-unit-borken",
		Type:           proto.UnitType_OUTPUT,
		ConfigStateIdx: 1,
		State:          proto.State_FAILED,
		LogLevel:       logLevel,
		Config: &proto.UnitExpectedConfig{
			Id:   "default",
			Type: "logstash",
			Name: "logstash",
			Source: integration.RequireNewStruct(t,
				map[string]interface{}{
					"type":    "logstash",
					"invalid": "configuration",
				}),
		},
	}

	// Those are the 'states' Filebeat will go through.
	// After each state is reached the mockServer will
	// send the next.
	protoUnits := [][]*proto.UnitExpected{
		{
			&healthyOutput,
			&filestreamInputHealthy,
		},
		{
			&brokenOutput,
			&filestreamInputStarting,
		},
		{
			&healthyOutput,
			&filestreamInputHealthy,
		},
		{}, // An empty one makes the Beat exit
	}

	// We use `success` to signal the test has ended successfully
	// if `success` is never closed, then the test will fail with a timeout.
	success := make(chan struct{})
	// The test is successful when we reach the last element of `protoUnits`
	onObserved := func(observed *proto.CheckinObserved, protoUnitsIdx int) {
		if protoUnitsIdx == len(protoUnits)-1 {
			close(success)
		}
	}

	server := integration.NewMockServer(
		protoUnits,
		[]uint64{0, 0, 0, 0},
		[]*proto.Features{nil, nil, nil, nil},
		onObserved,
		100*time.Millisecond,
	)
	require.NoError(t, server.Start(), "could not start the mock Elastic-Agent server")
	defer server.Stop()

	filebeat.RestartOnBeatOnExit = true
	filebeat.Start(
		"-E", fmt.Sprintf(`management.insecure_grpc_url_for_testing="localhost:%d"`, server.Port),
		"-E", "management.enabled=true",
		"-E", "management.restart_on_output_change=true",
	)

	select {
	case <-success:
	case <-time.After(60 * time.Second):
		t.Fatal("Output did not recover from a invalid configuration after 60s of waiting")
	}
}

// generateLogFile generates a log file by appending the current
// time to it every second.
func generateLogFile(t *testing.T, fullPath string) {
	t.Helper()
	f, err := os.Create(fullPath)
	if err != nil {
		t.Fatalf("could not create file '%s: %s", fullPath, err)
	}

	go func() {
		t.Helper()
		ticker := time.NewTicker(time.Second)
		t.Cleanup(ticker.Stop)

		done := make(chan struct{})
		t.Cleanup(func() { close(done) })

		defer func() {
			if err := f.Close(); err != nil {
				t.Errorf("could not close log file '%s': %s", fullPath, err)
			}
		}()

		for {
			select {
			case <-done:
				return
			case now := <-ticker.C:
				_, err := fmt.Fprintln(f, now.Format(time.RFC3339))
				if err != nil {
					// The Go compiler does not allow me to call t.Fatalf from a non-test
					// goroutine, so just log it instead
					t.Errorf("could not write data to log file '%s': %s", fullPath, err)
					return
				}
				// make sure log lines are synced as quickly as possible
				if err := f.Sync(); err != nil {
					t.Errorf("could not sync file '%s': %s", fullPath, err)
				}
			}
		}
	}()
}
