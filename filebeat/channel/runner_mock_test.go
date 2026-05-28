// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package channel

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"

	conf "github.com/elastic/elastic-agent-libs/config"
)

type runnerFactoryMock struct {
	clientCount int
	cfgs        []beat.ClientConfig
}

func (r *runnerFactoryMock) Create(p beat.PipelineConnector, config *conf.C) (cfgfile.Runner, error) {
	// When using the connector multiple times to create a client
	// it's using the same editor function for creating a new client
	// with a modified configuration that includes predefined processing.
	// This is why we must make sure nothing is re-used from one client to another.
	for i := 0; i < r.clientCount; i++ {
		client, err := p.ConnectWith(beat.ClientConfig{})
		if err != nil {
			return nil, err
		}

		// storing the config that the client was created with
		// it's needed for the `Assert` later
		r.cfgs = append(r.cfgs, client.(*clientMock).cfg) //nolint:errcheck //Safe to ignore in tests
	}
	return &noopRunner{}, nil
}

type noopRunner struct{}

func (*noopRunner) Start()         {}
func (*noopRunner) Stop()          {}
func (*noopRunner) String() string { return "noopRunner" }

func (runnerFactoryMock) CheckConfig(config *conf.C) error {
	return nil
}

// Assert runs various checks for the clients created by the wrapped
// pipeline connector. Processing.Meta and Processing.Fields must still be
// a per-client copy (they are mutated downstream); user-configured and
// index processors are shared across clients via noCloseProcessor wrappers
// (#50376) — see assertNoCloseProcessorsShared.
func (r runnerFactoryMock) Assert(t *testing.T) {
	t.Helper()

	// we need to make sure `Assert` is called after `Create`
	require.Len(t, r.cfgs, r.clientCount)

	sameBacking := func(a, b any) bool {
		return reflect.ValueOf(a).UnsafePointer() == reflect.ValueOf(b).UnsafePointer()
	}

	t.Run("new processing configuration each time", func(t *testing.T) {
		for i, c1 := range r.cfgs {
			for j, c2 := range r.cfgs {
				if i == j {
					continue
				}

				require.Falsef(t, sameBacking(c1.Processing.Meta, c2.Processing.Meta), "`Processing.Meta` cannot be re-used")
				require.Falsef(t, sameBacking(c1.Processing.Fields, c2.Processing.Fields), "`Processing.Fields` cannot be re-used")
			}
		}
	})

	t.Run("processor wrappers are per-client, but inner instances are shared", func(t *testing.T) {
		perClient := make([][]beat.Processor, 0, len(r.cfgs))
		for _, c := range r.cfgs {
			perClient = append(perClient, c.Processing.Processor.All())
			defer c.Processing.Processor.Close()
		}
		assertNoCloseProcessorsShared(t, perClient)
	})
}

type clientMock struct {
	cfg beat.ClientConfig
}

func (clientMock) Publish(beat.Event)      {}
func (clientMock) PublishAll([]beat.Event) {}
func (clientMock) Close() error            { return nil }

type pipelineConnectorMock struct{}

func (pipelineConnectorMock) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	client := &clientMock{
		cfg: cfg,
	}
	return client, nil
}

func (pipelineConnectorMock) Connect() (beat.Client, error) {
	return &clientMock{}, nil
}

func (pipelineConnectorMock) Disconnect(ctx context.Context) error {
	return nil
}
