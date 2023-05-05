// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package management

import (
	"testing"
	"time"

	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/stretchr/testify/require"
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
							"index":    "foo-index",
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
								"index":   "foo-index",
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
							"index":    "foo-index",
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
								"index":   "foo-index",
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
				t.Logf("done waiting, new state is %d", idx)
				return
			}
			return
		}
		waiting = true
		when = time.Now().Add(10 * time.Second)
	}
	server := &mock.StubServerV2{
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			t.Log("====================================================================================================")
			defer t.Log("====================================================================================================")
			t.Logf("[%s] Got %d units", time.Now().Format(time.RFC3339), len(observed.Units))
			if doesStateMatch(observed, units[idx], 0) {
				t.Logf("++++++++++ reached desired state, sending units[%d]", idx)
				nextState()
			}
			for i, unit := range observed.GetUnits() {
				t.Logf("Unit %d", i)
				t.Logf("ID %s, Type: %s, Message: %s, State %s, Payload %s",
					unit.GetId(),
					unit.GetType(),
					unit.GetMessage(),
					unit.GetState(),
					unit.GetPayload().String())

				if state := unit.GetState(); !(state == proto.State_HEALTHY || state != proto.State_CONFIGURING || state == proto.State_STARTING) {
					t.Fatalf("Unit '%s' is not healthy, state: %s", unit.GetId(), unit.GetState().String())
				}
			}
			return &proto.CheckinExpected{
				// AgentInfo:   agentInfo,
				// Features:    features[i],
				// FeaturesIdx: featuresIdxs[i],
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
	t.Logf("server started on port %d", server.Port)

	p := NewProc(t, "../../filebeat/filebeat.test", nil, server.Port)
	p.Start()
	t.Log("Filebeat started")

	p.LogContains("Can only start an input when all related states are finished", 2*time.Minute)        // centralmgmt
	p.LogContains("file 'flog.log' is not finished, will retry starting the input soon", 2*time.Minute) // centralmgmt.V2-manager
	p.LogContains("ForceReload set to TRUE", 2*time.Minute)                                             // centralmgmt.V2-manager
	p.LogContains("Reloading Beats inputs because forceReload is true", 2*time.Minute)                  // centralmgmt.V2-manager
	p.LogContains("ForceReload set to FALSE", 2*time.Minute)                                            // centralmgmt.V2-manager
	t.Log("********************************************* IT WORKS ****************************************************************************************************")
}
