// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package management

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

func TestPureServe(t *testing.T) {
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
								"paths":   []interface{}{"/tmp/flog.log"},
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
								"paths":   []interface{}{"/tmp/flog.log"},
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
			if doesStateMatch(observed, units[idx], 0) {
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

	p := NewProc(
		t,
		"../../filebeat.test",
		[]string{"-d",
			"centralmgmt, centralmgmt.V2-manager",
		},
		server.Port)
	p.Start()

	p.LogContains("Can only start an input when all related states are finished", 2*time.Minute)        // centralmgmt
	p.LogContains("file 'flog.log' is not finished, will retry starting the input soon", 2*time.Minute) // centralmgmt.V2-manager
	p.LogContains("ForceReload set to TRUE", 2*time.Minute)                                             // centralmgmt.V2-manager
	p.LogContains("Reloading Beats inputs because forceReload is true", 2*time.Minute)                  // centralmgmt.V2-manager
	p.LogContains("ForceReload set to FALSE", 2*time.Minute)                                            // centralmgmt.V2-manager
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
