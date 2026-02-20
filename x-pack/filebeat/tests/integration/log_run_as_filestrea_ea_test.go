// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

func TestLogAsFilestreamEA(t *testing.T) {
	filebeat := NewFilebeat(t)

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
						"id":                "run-as-filestream",
						"enabled":           true,
						"type":              "log",
						"paths":             []any{logfile},
						"run_as_filestream": true,
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
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
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

	// Wait for all events to be published
	filebeat.WaitPublishedEvents(5*time.Second, eventsCount)

	// Ensure events are from the correct input
	events := integration.GetEventsFromFileOutput[BeatEvent](filebeat, eventsCount, true)
	for i, ev := range events {
		if ev.Input.Type != "log" {
			t.Errorf("Event %d expecting type 'log', got %q", i, ev.Input.Type)
		}
		if !slices.Contains(ev.Tags, "take_over") {
			t.Errorf("Event %d does not contain 'take_over' tag", i)
		}
	}
}

func TestLogAsFilestreamContainerEA(t *testing.T) {
	filebeat := NewFilebeat(t)

	eventsCount := 50
	logDir := filepath.Join(filebeat.TempDir(), "containers")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("cannot create container logs directory: %s", err)
	}

	stdoutFile := filepath.Join(logDir, "container-stdout.log")
	stderrFile := filepath.Join(logDir, "container-stderr.log")
	integration.WriteDockerJSONLog(t, stdoutFile, eventsCount, []string{"stdout"}, false)
	integration.WriteDockerJSONLog(t, stderrFile, eventsCount, []string{"stderr"}, false)

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
		Id:             "container-input",
		Type:           proto.UnitType_INPUT,
		ConfigStateIdx: 1,
		State:          proto.State_HEALTHY,
		LogLevel:       proto.UnitLogLevel_DEBUG,
		Config: &proto.UnitExpectedConfig{
			Id:   "container-input",
			Type: "container",
			Name: "container",
			Streams: []*proto.Stream{
				{
					Id: "container-input",
					Source: integration.RequireNewStruct(t, map[string]any{
						"id":                   "run-as-filestream-container",
						"type":                 "container",
						"paths":                []any{filepath.Join(logDir, "*.log")},
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
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
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

	filebeat.WaitLogsContains(
		"Input 'filestream' starting",
		10*time.Second,
		"Filestream input did not start",
	)

	events := integration.GetEventsFromFileOutput[BeatEvent](filebeat, eventsCount*2, true)
	streamCounts := map[string]int{
		"stdout": 0,
		"stderr": 0,
	}
	for i, ev := range events {
		if ev.Input.Type != "container" {
			t.Errorf("Event %d expecting type 'container', got %q", i, ev.Input.Type)
		}

		if !strings.HasPrefix(ev.Message, "message ") {
			t.Errorf("Event %d: unexpected message %q", i, ev.Message)
		}

		if _, ok := streamCounts[ev.Stream]; !ok {
			t.Errorf("Event %d: unexpected stream %q", i, ev.Stream)
		} else {
			streamCounts[ev.Stream]++
		}

		if !slices.Contains(ev.Tags, "take_over") {
			t.Errorf("Event %d does not contain 'take_over' tag. %v", i, ev.Tags)
		}
	}

	if streamCounts["stdout"] != eventsCount {
		t.Errorf("expecting %d events from stdout, got %d", eventsCount, streamCounts["stdout"])
	}
	if streamCounts["stderr"] != eventsCount {
		t.Errorf("expecting %d events from stderr, got %d", eventsCount, streamCounts["stderr"])
	}
}

type BeatEvent struct {
	Input struct {
		Type string `json:"type"`
	} `json:"input"`
	Message string `json:"message"`
	Stream  string `json:"stream"`
	Log     struct {
		File struct {
			Fingerprint string `json:"fingerprint"`
		} `json:"file"`
	} `json:"log"`
	Tags []string
}
