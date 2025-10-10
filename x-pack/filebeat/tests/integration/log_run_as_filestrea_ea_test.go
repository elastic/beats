// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

func TestLogInputRunAsFilestreamEA(t *testing.T) {
	filebeat := NewFilebeat(t)
	finalStateReached := atomic.Bool{}

	eventsCount := 50
	logfile := filepath.Join(filebeat.TempDir(), "log.log")
	integration.WriteLogFile(t, logfile, eventsCount, false, "")

	output := proto.UnitExpected{
		Id:             "output-unit",
		Type:           proto.UnitType_OUTPUT,
		ConfigStateIdx: 1,
		State:          proto.State_HEALTHY,
		LogLevel:       proto.UnitLogLevel_DEBUG,
		Config: &proto.UnitExpectedConfig{
			Id:   "default",
			Type: "file",
			Name: "file",
			Source: integration.RequireNewStruct(t,
				map[string]any{
					"type":     "file",
					"path":     filebeat.TempDir(),
					"filename": "output-file",
				}),
		},
	}

	input := proto.UnitExpected{
		Id:             "log-input",
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
					Id: "log-input",
					Source: integration.RequireNewStruct(t, map[string]any{
						"id":                   "run-as-filestream",
						"enabled":              true,
						"type":                 "log",
						"paths":                []any{logfile},
						"run_as_filestream":    true,
						"allow_deprecated_use": true,
					}),
				},
			},
		},
	}

	units := []*proto.UnitExpected{
		&output,
		&input,
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
				finalStateReached.Store(true)
			}

			return &proto.CheckinExpected{
				Units: units,
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

	// Ensure the Filestream input is started
	filebeat.WaitLogsContains(
		"Input 'filestream' starting",
		10*time.Second,
		"Filestream input did not start",
	)

	events := integration.GetEventsFromFileOutput[BeatEvent](filebeat, eventsCount, true)
	for i, ev := range events {
		if ev.Input.Type != "log" {
			t.Errorf("Event %d expecting type 'log', got %q", i, ev.Input.Type)
		}
	}
}

type BeatEvent struct {
	Input struct {
		Type string `json:"type"`
	} `json:"input"`
	Log struct {
		File struct {
			Fingerprint string `json:"fingerprint"`
		} `json:"file"`
	} `json:"log"`
}
