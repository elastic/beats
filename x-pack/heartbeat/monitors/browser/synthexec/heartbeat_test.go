// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin || synthetics

package synthexec

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeartbeatRunner(t *testing.T) {
	runner := NewHeartbeatRunner(func() *SynthCmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestHeartbeatRunnerHelper", "--")
		cmd.Env = append(os.Environ(), "GO_WANT_HEARTBEAT_HELPER=1")
		return &SynthCmd{Cmd: cmd}
	})
	t.Cleanup(func() {
		_ = runner.Close()
	})

	ctx := context.WithValue(context.Background(), SynthexecTimeoutKey, time.Second)
	mpx, err := runner.run(ctx, heartbeatRunRequest{
		ID:   "run-1",
		Type: "run",
		Source: heartbeatSource{
			Type:   "inline",
			Script: "step('test', () => {})",
		},
	})
	require.NoError(t, err, "runner must start and accept a request")

	var eventTypes []string
	for event := range mpx.SynthEvents() {
		eventTypes = append(eventTypes, event.Type)
	}
	assert.Equal(t, []string{JourneyStart, CmdStatus}, eventTypes, "runner must forward events before closing the request stream")
}

func TestHeartbeatRunnerMultiplexesRequests(t *testing.T) {
	runner := NewHeartbeatRunner(func() *SynthCmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestHeartbeatRunnerHelper", "--")
		cmd.Env = append(os.Environ(), "GO_WANT_HEARTBEAT_HELPER=1")
		return &SynthCmd{Cmd: cmd}
	})
	t.Cleanup(func() {
		_ = runner.Close()
	})

	ctx := context.WithValue(context.Background(), SynthexecTimeoutKey, time.Second)
	first, err := runner.run(ctx, heartbeatRunRequest{ID: "run-1", Type: "run", Source: heartbeatSource{Type: "inline", Script: "step('first', () => {})"}})
	require.NoError(t, err, "first request must start")
	second, err := runner.run(ctx, heartbeatRunRequest{ID: "run-2", Type: "run", Source: heartbeatSource{Type: "inline", Script: "step('second', () => {})"}})
	require.NoError(t, err, "second request must start without waiting for the first")

	for _, mpx := range []*ExecMultiplexer{first, second} {
		var eventTypes []string
		for event := range mpx.SynthEvents() {
			eventTypes = append(eventTypes, event.Type)
		}
		assert.Equal(t, []string{JourneyStart, CmdStatus}, eventTypes, "each request must receive only its own events")
	}
}

func TestHeartbeatRunnerHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HEARTBEAT_HELPER") != "1" {
		return
	}

	control := os.NewFile(uintptr(heartbeatControlFD), "heartbeat-control")
	events := os.NewFile(uintptr(heartbeatEventFD), "heartbeat-events")
	controlEncoder := json.NewEncoder(control)
	eventEncoder := json.NewEncoder(events)
	if err := controlEncoder.Encode(heartbeatControlMessage{Type: "ready"}); err != nil {
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request heartbeatRunRequest
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			os.Exit(1)
		}
		event, err := json.Marshal(SynthEvent{Type: JourneyStart})
		if err != nil {
			os.Exit(1)
		}
		if err := eventEncoder.Encode(heartbeatEventEnvelope{ID: request.ID, Type: "heartbeat/event", Event: event}); err != nil {
			os.Exit(1)
		}
		if err := eventEncoder.Encode(heartbeatEventEnvelope{ID: request.ID, Type: heartbeatProtocolID}); err != nil {
			os.Exit(1)
		}
	}
}
