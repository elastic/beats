// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	lbmanagement "github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/x-pack/libbeat/management/tests"
	"github.com/elastic/beats/v7/x-pack/metricbeat/cmd"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestProcessStatusReporter(t *testing.T) {
	unitOneID := mock.NewID()
	unitOutID := mock.NewID()
	token := mock.NewID()

	tests.InitBeatsForTest(t, cmd.RootCmd)

	filename := fmt.Sprintf("test-%d", time.Now().Unix())
	outPath := filepath.Join(t.TempDir(), filename)
	t.Logf("writing output to file %s", outPath)
	err := os.Mkdir(outPath, 0775)
	require.NoError(t, err)
	defer func() {
		err := os.RemoveAll(outPath)
		require.NoError(t, err)
	}()

	// process with pid=-1 doesn't exist. This should degrade the input for a while
	inputStreamIncorrectPid := getInputStream(unitOneID, -1, 1)

	// process with valid pid. This should change state to healthy
	inputStreamCorrectPid := getInputStream(unitOneID, os.Getpid(), 2)

	outputExpectedStream := proto.UnitExpected{
		Id:             unitOutID,
		Type:           proto.UnitType_OUTPUT,
		ConfigStateIdx: 1,
		State:          proto.State_HEALTHY,
		Config: &proto.UnitExpectedConfig{
			DataStream: &proto.DataStream{
				Namespace: "default",
			},
			Type:     "file",
			Revision: 1,
			Meta: &proto.Meta{
				Package: &proto.Package{
					Name:    "system",
					Version: "1.17.0",
				},
			},
			Source: tests.RequireNewStruct(map[string]interface{}{
				"type":            "file",
				"enabled":         true,
				"path":            outPath,
				"filename":        "beat-out",
				"number_of_files": 7,
			}),
		},
	}

	observedStates := make(chan *proto.CheckinObserved)
	expectedUnits := make(chan []*proto.UnitExpected)
	done := make(chan struct{})

	server := &mock.StubServerV2{
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			select {
			case observedStates <- observed:
				return &proto.CheckinExpected{
					Units: <-expectedUnits,
				}
			case <-done:
				return nil
			}
		},
		ActionImpl: func(response *proto.ActionResponse) error {
			return nil
		},
	}
	require.NoError(t, server.Start(), "could not start V2 mock server")
	defer server.Stop()

	// start the client
	client := client.NewV2(fmt.Sprintf(":%d", server.Port), token, client.VersionInfo{
		Name: "program",
		Meta: map[string]string{
			"key": "value",
		},
	}, client.WithGRPCDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())))

	lbmanagement.SetManagerFactory(func(cfg *conf.C, registry *reload.Registry) (lbmanagement.Manager, error) {
		c := management.DefaultConfig()
		if err := cfg.Unpack(&c); err != nil {
			return nil, err
		}
		return management.NewV2AgentManagerWithClient(c, registry, client, management.WithStopOnEmptyUnits)
	})

	go func() {
		t.Logf("Running beats...")
		err := cmd.RootCmd.Execute()
		require.NoError(t, err)
	}()

	scenarios := []struct {
		expectedStatus proto.State
		nextInputunit  *proto.UnitExpected
	}{
		{
			proto.State_HEALTHY,
			&inputStreamIncorrectPid,
		},
		{
			proto.State_DEGRADED,
			&inputStreamCorrectPid,
		},
		{
			proto.State_HEALTHY,
			&inputStreamCorrectPid,
		},
		// wait for one more checkin, just to be sure it's healthy
		{
			proto.State_HEALTHY,
			&inputStreamCorrectPid,
		},
	}

	timeout := 2 * time.Minute
	timer := time.NewTimer(timeout)

	for id := 0; id < len(scenarios); {
		select {
		case observed := <-observedStates:
			state := extractState(observed.GetUnits(), unitOneID)
			expectedUnits <- []*proto.UnitExpected{
				scenarios[id].nextInputunit,
				&outputExpectedStream,
			}
			if state != scenarios[id].expectedStatus {
				continue
			}
			// always ensure that output is healthy
			outputState := extractState(observed.GetUnits(), unitOutID)
			require.Equal(t, outputState, proto.State_HEALTHY)

			timer.Reset(timeout)
			id++
		case <-timer.C:
			t.Fatalf("timeout after %s waiting for checkin", timeout)
		default:
		}
	}
}

func extractState(units []*proto.UnitObserved, idx string) proto.State {
	for _, unit := range units {
		if unit.Id == idx {
			return unit.GetState()
		}
	}
	return -1
}

func getInputStream(id string, pid int, stateIdx int) proto.UnitExpected {
	return proto.UnitExpected{
		Id:             id,
		Type:           proto.UnitType_INPUT,
		ConfigStateIdx: uint64(stateIdx),
		State:          proto.State_HEALTHY,
		Config: &proto.UnitExpectedConfig{
			DataStream: &proto.DataStream{
				Namespace: "default",
			},
			Streams: []*proto.Stream{{
				Id: "system/metrics-system.process-default-system",
				DataStream: &proto.DataStream{
					Dataset: "system.process",
					Type:    "metrics",
				},
				Source: tests.RequireNewStruct(map[string]interface{}{
					"metricsets":  []interface{}{"process"},
					"process.pid": pid,
				}),
			}},
			Type:     "system/metrics",
			Id:       "system/metrics-system-default-system",
			Name:     "system-1",
			Revision: 1,
			Meta: &proto.Meta{
				Package: &proto.Package{
					Name:    "system",
					Version: "1.17.0",
				},
			},
		},
	}
}
