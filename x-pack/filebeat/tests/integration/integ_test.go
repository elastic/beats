// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package input

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

// TestInputReloadUnderElasticAgent will start a Filebeat and cause the input
// reload issue described on https://github.com/elastic/beats/issues/33653
// to happen, then it will check to logs to ensure the fix is working.
//
// In case of a test failure the directory with Filebeat logs and
// all other supporting files will be kept on build/integration-tests.
//
// Run the tests wit -v flag to print the temporary folder used.
func TestInputReloadUnderElasticAgent(t *testing.T) {
	// We create our own temp dir so the files can be persisted
	// in case the test fails. This will help debugging issues on CI
	//
	// testFailed will be set to 'false' as the very last thing on this test,
	// it allows us to use t.CleanUp to remove the temporary files
	testFailed := true
	tempDir, err := filepath.Abs(filepath.Join("../../build/integration-tests/",
		fmt.Sprintf("%s-%d", t.Name(), time.Now().Unix())))
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(tempDir, 0766); err != nil {
		t.Fatalf("cannot create tmp dir: %#v, msg: %s", err, err.Error())
	}
	t.Cleanup(func() {
		if !testFailed {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Fatalf("could not remove temp dir '%s': %s", tempDir, err)
			}
			t.Logf("Temprary directory '%s' removed", tempDir)
		}
	})

	t.Logf("Temporary directory: %s", tempDir)

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
		ActionImpl: func(response *proto.ActionResponse) error {
			return nil
		},
		ActionsChan: make(chan *mock.PerformAction, 100),
	}

	require.NoError(t, server.Start())
	defer server.Stop()

	p := NewBeat(
		t,
		"../../filebeat.test",
		[]string{
			"-E", fmt.Sprintf("management.insecure_grpc_url_for_testing=\"localhost:%d\"", server.Port),
			"-E", "management.enabled=true",
		},
		tempDir,
	)

	p.Start()

	p.LogContains("Can only start an input when all related states are finished", 5*time.Minute)        // logger: centralmgmt
	p.LogContains("file 'flog.log' is not finished, will retry starting the input soon", 5*time.Minute) // logger: centralmgmt.V2-manager
	p.LogContains("ForceReload set to TRUE", 5*time.Minute)                                             // logger: centralmgmt.V2-manager
	p.LogContains("Reloading Beats inputs because forceReload is true", 5*time.Minute)                  // logger: centralmgmt.V2-manager
	p.LogContains("ForceReload set to FALSE", 5*time.Minute)                                            // logger: centralmgmt.V2-manager

	// Set it to false, so the temporaty directory is removed
	testFailed = false
}

func doesStateMatch(
	observed *proto.CheckinObserved,
	expectedUnits []*proto.UnitExpected,
	expectedFeaturesIdx uint64,
) bool {
	if len(observed.Units) != len(expectedUnits) {
		return false
	}
	expectedMap := make(map[unitKey]*proto.UnitExpected)
	for _, exp := range expectedUnits {
		expectedMap[unitKey{client.UnitType(exp.Type), exp.Id}] = exp
	}
	for _, unit := range observed.Units {
		exp, ok := expectedMap[unitKey{client.UnitType(unit.Type), unit.Id}]
		if !ok {
			return false
		}
		if unit.State != exp.State || unit.ConfigStateIdx != exp.ConfigStateIdx {
			return false
		}
	}

	if observed.FeaturesIdx != expectedFeaturesIdx {
		return false
	}

	return true
}

type unitKey struct {
	Type client.UnitType
	ID   string
}

func requireNewStruct(t *testing.T, v map[string]interface{}) *structpb.Struct {
	str, err := structpb.NewStruct(v)
	if err != nil {
		require.NoError(t, err)
	}
	return str
}

func generateLogFile(t *testing.T, fullPath string) {
	t.Helper()
	f, err := os.Create(fullPath)
	if err != nil {
		t.Fatalf("could not create file '%s: %s", fullPath, err)
	}

	go func() {
		t.Helper()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		defer f.Close()
		for {
			now := <-ticker.C
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
	}()
}
