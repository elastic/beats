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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/libbeat/common/reload"
)

func TestManagerV2(t *testing.T) {

	srv := mockSrvWithUnits([]*proto.UnitExpected{
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
	})
	require.NoError(t, srv.Start())
	defer srv.Stop()

	client := client.NewV2(fmt.Sprintf(":%d", srv.Port), "", client.VersionInfo{
		Name:    "program",
		Version: "v1.0.0",
		Meta: map[string]string{
			"key": "value",
		},
	}, grpc.WithTransportCredentials(insecure.NewCredentials()))

	r := reload.NewRegistry()

	output := &reloadable{}
	r.MustRegisterOutput(output)
	inputs := &reloadableList{}
	r.MustRegisterInput(inputs)

	m, err := NewV2AgentManagerWithClient(&Config{
		Enabled: true,
	}, r, client)
	require.NoError(t, err)

	err = m.Start()
	require.NoError(t, err)
	defer m.Stop()

	require.Eventually(t, func() bool {
		outputCfg := output.Config()
		inputCfgs := inputs.Configs()
		return outputCfg != nil && len(inputCfgs) == 3
	}, 5*time.Second, 100*time.Millisecond)
}

func mockSrvWithUnits(units []*proto.UnitExpected) *mock.StubServerV2 {
	return &mock.StubServerV2{
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			return &proto.CheckinExpected{
				AgentInfo: &proto.CheckinAgentInfo{
					Id:       "elastic-agent-id",
					Version:  "8.6.0",
					Snapshot: true,
				},
				Units: units,
			}
		},
		ActionImpl: func(response *proto.ActionResponse) error {
			// actions not tested here
			return nil
		},
		ActionsChan: make(chan *mock.PerformAction, 100),
	}
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
