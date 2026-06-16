// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

func TestUDPReportsError(t *testing.T) {
	filebeat := NewFilebeat(t)

	// get a random port on all interfaces
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("cannot create UDPAddr: %s", err)
	}

	ln, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("cannot create UDP listener: %s", err)
	}

	t.Cleanup(func() { ln.Close() })

	udpInput := proto.UnitExpected{
		Id:             "input",
		Type:           proto.UnitType_INPUT,
		ConfigStateIdx: 1,
		State:          proto.State_FAILED,
		LogLevel:       proto.UnitLogLevel_DEBUG,
		Config: &proto.UnitExpectedConfig{
			Id:   "udp-input",
			Type: "udp",
			Name: "udp",
			Streams: []*proto.Stream{
				{
					Id: "udp-input",
					Source: integration.RequireNewStruct(t, map[string]any{
						"enabled": true,
						"type":    "udp",
						"host":    ln.LocalAddr().String(),
					}),
				},
			},
		},
	}

	output := proto.UnitExpected{
		Id:             "output-unit",
		Type:           proto.UnitType_OUTPUT,
		ConfigStateIdx: 1,
		State:          proto.State_HEALTHY,
		LogLevel:       proto.UnitLogLevel_DEBUG,
		Config: &proto.UnitExpectedConfig{
			Id:   "default",
			Type: "discard",
			Name: "discard",
			Source: integration.RequireNewStruct(t,
				map[string]any{
					"type":    "discard",
					"enabled": true,
				}),
		},
	}

	units := []*proto.UnitExpected{
		&output,
		&udpInput,
	}

	errMsgCh := make(chan string)
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)

	server := &mock.StubServerV2{
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			if management.DoesStateMatch(observed, units, 0) {
				for _, unit := range observed.Units {
					if unit.GetId() == udpInput.GetId() {
						errMsgCh <- unit.GetMessage()
					}
				}
			}

			return &proto.CheckinExpected{
				Units: units,
			}
		},
		ActionImpl: func(response *proto.ActionResponse) error { return nil },
	}

	if err := server.Start(); err != nil {
		t.Fatalf("could not start V2 server: %s", err)
	}
	t.Cleanup(server.Stop)

	filebeat.Start(
		"-E", fmt.Sprintf(`management.insecure_grpc_url_for_testing="localhost:%d"`, server.Port),
		"-E", "management.enabled=true",
	)

	select {
	case <-ctx.Done():
		t.Fatal("timed out while waiting input unit to be unhealthy")
	case msg := <-errMsgCh:
		if !strings.Contains(msg, "bind: address already in use") {
			t.Fatalf("did not find the expected error message, got: '%s'", msg)
		}
	}
}
