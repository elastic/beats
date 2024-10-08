// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

func TestSimpleInputConfig(t *testing.T) {
	// Uncomment the line below to see the debug logs for this test
	// logp.DevelopmentSetup(logp.WithLevel(logp.DebugLevel), logp.WithSelectors("*"))
	r := reload.NewRegistry()

	output := &mockOutput{
		ReloadFn: func(config *reload.ConfigWithMeta) error {
			return nil
		},
	}
	r.MustRegisterOutput(output)
	inputs := &mockReloadable{
		ReloadFn: func(configs []*reload.ConfigWithMeta) error {
			return nil
		},
	}
	r.MustRegisterInput(inputs)

	stateReached := atomic.Bool{}
	units := []*proto.UnitExpected{
		{
			Id:             "output-unit",
			Type:           proto.UnitType_OUTPUT,
			State:          proto.State_HEALTHY,
			ConfigStateIdx: 1,
			LogLevel:       proto.UnitLogLevel_DEBUG,
			Config: &proto.UnitExpectedConfig{
				Id:   "default",
				Type: "mock",
				Name: "mock",
				Source: integration.RequireNewStruct(t,
					map[string]interface{}{
						"Is":        "this",
						"required?": "Yes!",
					}),
			},
		},
		{
			Id:             "input-unit",
			Type:           proto.UnitType_INPUT,
			State:          proto.State_HEALTHY,
			ConfigStateIdx: 1,
			LogLevel:       proto.UnitLogLevel_DEBUG,
			Config: &proto.UnitExpectedConfig{
				Id:   "the-id-in-the-input-config",
				Type: "filestream",
				// All fields get repeated here, including ID.
				Source: integration.RequireNewStruct(t,
					map[string]interface{}{
						"paths": []any{
							"/tmp/logfile.log",
						},
					},
				),
			},
		},
	}

	desiredState := []*proto.UnitExpected{
		{
			Id:             "output-unit",
			Type:           proto.UnitType_OUTPUT,
			State:          proto.State_HEALTHY,
			ConfigStateIdx: 1,
			LogLevel:       proto.UnitLogLevel_DEBUG,
			Config: &proto.UnitExpectedConfig{
				Id:   "default",
				Type: "filestream",
				Name: "mock",
				Source: integration.RequireNewStruct(t,
					map[string]interface{}{
						"this":     "is",
						"required": true,
					}),
			},
		},
		{
			Id:             "input-unit",
			Type:           proto.UnitType_INPUT,
			State:          proto.State_HEALTHY,
			ConfigStateIdx: 1,
			LogLevel:       proto.UnitLogLevel_DEBUG,
		},
	}

	server := &mock.StubServerV2{
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			// If the desired state has been reached, return nil
			// so the manager can shutdown.
			if stateReached.Load() {
				return nil
			}

			if DoesStateMatch(observed, desiredState, 0) {
				stateReached.Store(true)
			}
			return &proto.CheckinExpected{
				Units: units,
			}
		},
		ActionImpl: func(response *proto.ActionResponse) error { return nil },
	}

	if err := server.Start(); err != nil {
		t.Fatalf("could not start mock Elastic-Agent server: %s", err)
	}
	defer server.Stop()

	client := client.NewV2(
		fmt.Sprintf(":%d", server.Port),
		"",
		client.VersionInfo{},
		client.WithGRPCDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())))

	m, err := NewV2AgentManagerWithClient(
		&Config{
			Enabled: true,
		},
		r,
		client,
	)
	if err != nil {
		t.Fatalf("could not instantiate ManagerV2: %s", err)
	}

	if err := m.Start(); err != nil {
		t.Fatalf("could not start ManagerV2: %s", err)
	}
	defer m.Stop()

	require.Eventually(t, func() bool {
		return stateReached.Load()
	}, 10*time.Second, 100*time.Millisecond, "desired state, output failed, was not reached")
}
