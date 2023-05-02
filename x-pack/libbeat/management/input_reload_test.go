// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

func TestInputReload(t *testing.T) {
	// Uncomment this line to see the debug logs for this test
	// logp.DevelopmentSetup(logp.WithLevel(logp.DebugLevel), logp.WithSelectors("*", "centralmgmt.V2-manager"))
	r := reload.NewRegistry()

	output := &reloadable{}
	r.MustRegisterOutput(output)

	reloadCallCount := 0
	inputs := &reloadableListMock{
		ReloadImpl: func(configs []*reload.ConfigWithMeta) error {
			reloadCallCount++
			if reloadCallCount == 1 {
				e1 := multierror.Errors{withCause{&common.ErrInputNotFinished{
					State: "<state string goes here>",
					File:  "/tmp/foo.log",
				}}}
				return e1.Err()
			}

			return nil
		},
	}
	r.MustRegisterInput(inputs)

	configIdx := -1
	onObserved := func(observed *proto.CheckinObserved, currentIdx int) {
		configIdx = currentIdx
	}

	srv := mockSrv([][]*proto.UnitExpected{
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
								"paths": []interface{}{"/tmp/foo.log"},
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
				},
			},
			{
				Id:             "input-unit-1",
				Type:           proto.UnitType_INPUT,
				ConfigStateIdx: 1,
				State:          proto.State_HEALTHY,
				LogLevel:       proto.UnitLogLevel_DEBUG,
				Config: &proto.UnitExpectedConfig{
					Id:   "log-input-2",
					Type: "log",
					Name: "log",
					Streams: []*proto.Stream{
						{
							Id: "log-input-2",
							Source: requireNewStruct(t, map[string]interface{}{
								"paths": []interface{}{"/tmp/foo.log"},
							}),
						},
					},
				},
			},
		},
	},
		[]uint64{1, 1},
		[]*proto.Features{
			nil,
			nil,
		},
		onObserved,
		500*time.Millisecond,
	)
	require.NoError(t, srv.Start())
	defer srv.Stop()

	client := client.NewV2(fmt.Sprintf(":%d", srv.Port), "", client.VersionInfo{
		Name:    "program",
		Version: "v1.0.0",
		Meta: map[string]string{
			"key": "value",
		},
	}, grpc.WithTransportCredentials(insecure.NewCredentials()))

	m, err := NewV2AgentManagerWithClient(&Config{
		Enabled: true,
	}, r, client)
	require.NoError(t, err)

	mm := m.(*BeatV2Manager)

	err = m.Start()
	require.NoError(t, err)
	defer m.Stop()

	forceReloadFalseCount := 0
	forceReloadState := []bool{false, true, false}
	forceReloadStateIdx := 0
	forceReloadLastState := true // starts on true so the first iteratiction is already a change

	// TODO: write a state machine to check the ForceReload Flip alongside the config indes (from the observe callback)
	require.Eventually(t, func() bool {
		forceReload := mm.forceReload.Load()
		// That detects a state change, we only count/advance steps
		// on state changes
		if forceReload != forceReloadLastState {
			forceReloadLastState = forceReload
			if forceReload == forceReloadState[forceReloadStateIdx] {
				// Setp to the next state
				forceReloadStateIdx++
			}

			// If we went through all states, then succeed
			if forceReloadStateIdx == len(forceReloadState) {
				// If we went through all states
				if configIdx == 1 {
					return true
				}
			}
		}

		return configIdx == 1 && forceReloadFalseCount == 2
	}, 15*time.Hour, 300*time.Millisecond)
}

type reloadableListMock struct {
	mx         sync.Mutex
	configs    []*reload.ConfigWithMeta
	ReloadImpl func(configs []*reload.ConfigWithMeta) error
}

func (r *reloadableListMock) Reload(configs []*reload.ConfigWithMeta) error {
	r.mx.Lock()
	defer r.mx.Unlock()
	return r.ReloadImpl(configs)
}

func (r *reloadableListMock) Configs() []*reload.ConfigWithMeta {
	r.mx.Lock()
	defer r.mx.Unlock()
	return r.configs
}

type withCause struct {
	Err error
}

func (w withCause) Cause() error {
	return w.Err
}

func (w withCause) Error() string {
	return w.Err.Error()
}
