// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tests

import (
	"fmt"
	"sync"
	"testing"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

// MockV2Handler wraps the basic tooling needed to handle a fake V2 controller
type MockV2Handler struct {
	Srv    mock.StubServerV2
	Client client.V2
}

// NewMockServer returns a mocked elastic-agent V2 controller
func NewMockServer(t *testing.T, canStop func(string) bool, inputConfig *proto.UnitExpectedConfig, outPath string) MockV2Handler {
	unitOneID := mock.NewID()
	unitOutID := mock.NewID()

	token := mock.NewID()
	// var gotConfig bool

	var mut sync.Mutex

	var logOutputStream = &proto.UnitExpectedConfig{
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
		Source: RequireNewStruct(map[string]interface{}{
			"type":            "file",
			"enabled":         true,
			"path":            outPath,
			"filename":        "beat-out",
			"number_of_files": 7,
		}),
	}

	stopping := false
	srv := mock.StubServerV2{
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			mut.Lock()
			defer mut.Unlock()
			if observed.Token == token {
				// initial checkin
				if !stopping && (len(observed.Units) == 0 || observed.Units[0].State == proto.State_STARTING) {
					return sendUnitsWithState(proto.State_HEALTHY, inputConfig, logOutputStream, unitOneID, unitOutID, 1)
				} else if !stopping && checkUnitStateHealthy(observed.Units) {
					if canStop(outPath) {
						// remove the units once the callback says we can
						stopping = true
						return sendUnitsWithState(proto.State_STOPPED, inputConfig, logOutputStream, unitOneID, unitOutID, 1)
					}
					// we still want them healthy
					return sendUnitsWithState(proto.State_HEALTHY, inputConfig, logOutputStream, unitOneID, unitOutID, 1)
				} else if stopping {
					if len(observed.Units) == 0 {
						return &proto.CheckinExpected{}
					}
					if observed.Units[0].State != proto.State_STOPPED {
						// keep telling them to stop
						return sendUnitsWithState(proto.State_STOPPED, inputConfig, logOutputStream, unitOneID, unitOutID, 1)
					}
					// all units have now stopped, can be removed
					return &proto.CheckinExpected{}
				}
			}
			return &proto.CheckinExpected{}
		},
		ActionImpl: func(response *proto.ActionResponse) error {
			return nil
		},
		ActionsChan: make(chan *mock.PerformAction, 100),
	} // end of srv declaration

	// The start() needs to happen here, since the client needs the assigned server port
	err := srv.Start()
	require.NoError(t, err)

	client := client.NewV2(fmt.Sprintf(":%d", srv.Port), token, client.VersionInfo{
		Name: "program",
		Meta: map[string]string{
			"key": "value",
		},
	}, client.WithGRPCDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())))

	return MockV2Handler{Srv: srv, Client: client}
}

// helper to wrap the CheckinExpected config we need with every refresh of the mock server
func sendUnitsWithState(state proto.State, input, output *proto.UnitExpectedConfig, inId, outId string, stateIndex uint64) *proto.CheckinExpected {
	return &proto.CheckinExpected{
		AgentInfo: &proto.AgentInfo{
			Id:       "test-agent",
			Version:  "8.4.0",
			Snapshot: true,
		},
		Units: []*proto.UnitExpected{
			{
				Id:             inId,
				Type:           proto.UnitType_INPUT,
				ConfigStateIdx: stateIndex,
				Config:         input,
				State:          state,
				LogLevel:       proto.UnitLogLevel_DEBUG,
			},
			{
				Id:             outId,
				Type:           proto.UnitType_OUTPUT,
				ConfigStateIdx: stateIndex,
				Config:         output,
				State:          state,
			},
		},
	}
}

func checkUnitStateHealthy(units []*proto.UnitObserved) bool {
	if len(units) == 0 {
		return false
	}
	for _, unit := range units {
		if unit.State != proto.State_HEALTHY {
			return false
		}
	}
	return true
}

// RequireNewStruct converts a mapstr to a protobuf struct
func RequireNewStruct(v map[string]interface{}) *structpb.Struct {
	str, err := structpb.NewStruct(v)
	if err != nil {
		panic(err)
	}
	return str
}
