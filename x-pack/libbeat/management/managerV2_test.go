// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/libbeat/common/reload"
)

func TestManagerV2(t *testing.T) {

	r := reload.NewRegistry()

	output := &reloadable{}
	r.MustRegisterOutput(output)
	inputs := &reloadableList{}
	r.MustRegisterInput(inputs)

	configsSet := false
	configsCleared := false
	logLevelSet := false
	allStopped := false
	onObserved := func(observed *proto.CheckinObserved, currentIdx int) {
		if currentIdx == 1 {
			oCfg := output.Config()
			iCfgs := inputs.Configs()
			if oCfg != nil && len(iCfgs) == 3 {
				configsSet = true
				t.Logf("output and inputs configuration set")
			}
		} else if currentIdx == 2 {
			oCfg := output.Config()
			iCfgs := inputs.Configs()
			if oCfg == nil || len(iCfgs) != 3 {
				// should not happen (config no longer set)
				configsSet = false
				t.Logf("output and inputs configuration cleared (should not happen)")
			}
		} else {
			oCfg := output.Config()
			iCfgs := inputs.Configs()
			if oCfg == nil && len(iCfgs) == 0 {
				configsCleared = true
			}
			if len(observed.Units) == 0 {
				allStopped = true
				t.Logf("output and inputs configuration cleared (stopping)")
			}
		}
		if logp.GetLevel() == zapcore.DebugLevel {
			logLevelSet = true
			t.Logf("debug log level set")
		}
	}

	srv := mockSrvWithUnits([][]*proto.UnitExpected{
		{
			{
				Id:             "output-unit",
				Type:           proto.UnitType_OUTPUT,
				ConfigStateIdx: 1,
				Config: &proto.UnitExpectedConfig{
					Id:   "default",
					Type: "elasticsearch",
					Name: "elasticsearch",
				},
				State:    proto.State_HEALTHY,
				LogLevel: proto.UnitLogLevel_INFO,
			},
			{
				Id:             "input-unit-1",
				Type:           proto.UnitType_INPUT,
				ConfigStateIdx: 1,
				Config: &proto.UnitExpectedConfig{
					Id:   "system/metrics-system-default-system-1",
					Type: "system/metrics",
					Name: "system-1",
					Streams: []*proto.Stream{
						{
							Id: "system/metrics-system.filesystem-default-system-1",
							Source: requireNewStruct(t, map[string]interface{}{
								"metricsets": []interface{}{"filesystem"},
								"period":     "1m",
							}),
						},
					},
				},
				State:    proto.State_HEALTHY,
				LogLevel: proto.UnitLogLevel_INFO,
			},
			{
				Id:             "input-unit-2",
				Type:           proto.UnitType_INPUT,
				ConfigStateIdx: 1,
				Config: &proto.UnitExpectedConfig{
					Id:   "system/metrics-system-default-system-2",
					Type: "system/metrics",
					Name: "system-2",
					Streams: []*proto.Stream{
						{
							Id: "system/metrics-system.filesystem-default-system-2",
							Source: requireNewStruct(t, map[string]interface{}{
								"metricsets": []interface{}{"filesystem"},
								"period":     "1m",
							}),
						},
						{
							Id: "system/metrics-system.filesystem-default-system-3",
							Source: requireNewStruct(t, map[string]interface{}{
								"metricsets": []interface{}{"filesystem"},
								"period":     "1m",
							}),
						},
					},
				},
				State:    proto.State_HEALTHY,
				LogLevel: proto.UnitLogLevel_INFO,
			},
		},
		{
			{
				Id:             "output-unit",
				Type:           proto.UnitType_OUTPUT,
				ConfigStateIdx: 1,
				State:          proto.State_HEALTHY,
				LogLevel:       proto.UnitLogLevel_INFO,
			},
			{
				Id:             "input-unit-1",
				Type:           proto.UnitType_INPUT,
				ConfigStateIdx: 1,
				State:          proto.State_HEALTHY,
				LogLevel:       proto.UnitLogLevel_DEBUG,
			},
			{
				Id:             "input-unit-2",
				Type:           proto.UnitType_INPUT,
				ConfigStateIdx: 1,
				State:          proto.State_HEALTHY,
				LogLevel:       proto.UnitLogLevel_INFO,
			},
		},
		{
			{
				Id:             "output-unit",
				Type:           proto.UnitType_OUTPUT,
				ConfigStateIdx: 1,
				State:          proto.State_STOPPED,
				LogLevel:       proto.UnitLogLevel_INFO,
			},
			{
				Id:             "input-unit-1",
				Type:           proto.UnitType_INPUT,
				ConfigStateIdx: 1,
				State:          proto.State_STOPPED,
				LogLevel:       proto.UnitLogLevel_DEBUG,
			},
			{
				Id:             "input-unit-2",
				Type:           proto.UnitType_INPUT,
				ConfigStateIdx: 1,
				State:          proto.State_STOPPED,
				LogLevel:       proto.UnitLogLevel_INFO,
			},
		},
		{},
	}, onObserved, 500*time.Millisecond)
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

	err = m.Start()
	require.NoError(t, err)
	defer m.Stop()

	require.Eventually(t, func() bool {
		return configsSet && configsCleared && logLevelSet && allStopped
	}, 15*time.Second, 300*time.Millisecond)
}

func mockSrvWithUnits(units [][]*proto.UnitExpected, observedCallback func(*proto.CheckinObserved, int), delay time.Duration) *mock.StubServerV2 {
	i := 0
	agentInfo := &proto.CheckinAgentInfo{
		Id:       "elastic-agent-id",
		Version:  "8.6.0",
		Snapshot: true,
	}
	return &mock.StubServerV2{
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			if observedCallback != nil {
				observedCallback(observed, i)
			}
			matches := doesStateMatch(observed, units[i])
			if !matches {
				// send same set of units
				return &proto.CheckinExpected{
					AgentInfo: agentInfo,
					Units:     units[i],
				}
			}
			// delay sending next expected based on delay
			if delay > 0 {
				<-time.After(delay)
			}
			// send next set of units
			i += 1
			if i >= len(units) {
				// stay on last index
				i = len(units) - 1
			}
			return &proto.CheckinExpected{
				AgentInfo: agentInfo,
				Units:     units[i],
			}
		},
		ActionImpl: func(response *proto.ActionResponse) error {
			// actions not tested here
			return nil
		},
		ActionsChan: make(chan *mock.PerformAction, 100),
	}
}

func doesStateMatch(observed *proto.CheckinObserved, expected []*proto.UnitExpected) bool {
	if len(observed.Units) != len(expected) {
		return false
	}
	expectedMap := make(map[unitKey]*proto.UnitExpected)
	for _, exp := range expected {
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
	return true
}

type reloadable struct {
	mx     sync.Mutex
	config *reload.ConfigWithMeta
}

type reloadableList struct {
	mx      sync.Mutex
	configs []*reload.ConfigWithMeta
}

func (r *reloadable) Reload(config *reload.ConfigWithMeta) error {
	r.mx.Lock()
	defer r.mx.Unlock()
	r.config = config
	return nil
}

func (r *reloadable) Config() *reload.ConfigWithMeta {
	r.mx.Lock()
	defer r.mx.Unlock()
	return r.config
}

func (r *reloadableList) Reload(configs []*reload.ConfigWithMeta) error {
	r.mx.Lock()
	defer r.mx.Unlock()
	r.configs = configs
	return nil
}

func (r *reloadableList) Configs() []*reload.ConfigWithMeta {
	r.mx.Lock()
	defer r.mx.Unlock()
	return r.configs
}
