// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

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
	ensureESIsRunning(t)

	// We create our own temp dir so the files can be persisted
	// in case the test fails. This will help debugging issues
	// locally and on CI.
	//
	// testSucceeded will be set to 'true' as the very last thing on this test,
	// it allows us to use t.CleanUp to remove the temporary files
	testSucceeded := false
	tempDir, err := filepath.Abs(filepath.Join("../../build/integration-tests/",
		fmt.Sprintf("%s-%d", t.Name(), time.Now().Unix())))
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(tempDir, 0766); err != nil {
		t.Fatalf("cannot create tmp dir: %s, msg: %s", err, err.Error())
	}
	t.Logf("Temporary directory: %s", tempDir)
	t.Cleanup(func() {
		if testSucceeded {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Fatalf("could not remove temp dir '%s': %s", tempDir, err)
			}
			t.Logf("Temporary directory '%s' removed", tempDir)
		}
	})

	logFilePath := filepath.Join(tempDir, "flog.log")
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
					Source: requireNewStruct(t,
						map[string]interface{}{
							"type":     "elasticsearch",
							"hosts":    []interface{}{"http://localhost:9200"},
							"username": "admin",
							"password": "testing",
							"protocol": "http",
							"enabled":  true,
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
							Source: requireNewStruct(t, map[string]interface{}{
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
					Source: requireNewStruct(t,
						map[string]interface{}{
							"type":     "elasticsearch",
							"hosts":    []interface{}{"http://localhost:9200"},
							"username": "admin",
							"password": "testing",
							"protocol": "http",
							"enabled":  true,
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
							Source: requireNewStruct(t, map[string]interface{}{
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

	filebeat := NewBeat(
		t,
		"../../filebeat.test",
		tempDir,
		"-E", fmt.Sprintf(`management.insecure_grpc_url_for_testing="localhost:%d"`, server.Port),
		"-E", "management.enabled=true",
	)

	filebeat.Start()

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

	// Set it to true, so the temporary directory is removed
	testSucceeded = true
}

func requireNewStruct(t *testing.T, v map[string]interface{}) *structpb.Struct {
	str, err := structpb.NewStruct(v)
	if err != nil {
		require.NoError(t, err)
	}
	return str
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

func ensureESIsRunning(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(500*time.Second))
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:9200", nil)
	if err != nil {
		t.Fatalf("cannot create request to ensure ES is running: %s", err)
	}

	user := os.Getenv("ES_USER")
	if user == "" {
		user = "admin"
	}

	pass := os.Getenv("ES_PASS")
	if pass == "" {
		pass = "testing"
	}

	req.SetBasicAuth(user, pass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// If you're reading this message, you probably forgot to start ES
		// run `mage compose:Up` from Filebeat's folder to start all
		// containers required for integration tests
		t.Fatalf("cannot execute HTTP request to ES: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected HTTP status: %d, expecting 200 - OK", resp.StatusCode)
	}
}
