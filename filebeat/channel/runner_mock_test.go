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
	"reflect"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"

	conf "github.com/elastic/elastic-agent-libs/config"

	"github.com/stretchr/testify/require"
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

// Assert runs various checks for the clients created by the wrapped pipeline connector.
//
// `Processing.Meta` and `Processing.Fields` must still be a per-client copy —
// these are mutated downstream and sharing them would let one client see
// another's metadata.
//
// User-configured processors and the index processor, however, are now built
// once per input and shared across clients via noCloseProcessor wrappers. We
// verify both: the wrappers must be distinct per client (so closing one
// client's list does not affect siblings) AND the underlying inner instances
// must be the same object (the whole point of the fix — see
// elastic/beats#50376).
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
		var firstList []beat.Processor
		for idx, c := range r.cfgs {
			list := c.Processing.Processor.All()
			require.NotEmptyf(t, list, "client %d processor list cannot be empty", idx)
			defer c.Processing.Processor.Close()

			if idx == 0 {
				firstList = list
				continue
			}

			require.Lenf(t, list, len(firstList), "client %d processor list length differs from client 0", idx)
			for j := range list {
				w0, ok0 := firstList[j].(*noCloseProcessor)
				wi, oki := list[j].(*noCloseProcessor)
				require.Truef(t, ok0, "client 0 processor[%d] expected *noCloseProcessor, got %T", j, firstList[j])
				require.Truef(t, oki, "client %d processor[%d] expected *noCloseProcessor, got %T", idx, j, list[j])
				require.NotSamef(t, w0, wi, "client %d processor[%d]: wrappers must be distinct allocations so per-client Close is isolated", idx, j)
				require.Samef(t, w0.inner, wi.inner, "client %d processor[%d]: inner shared instance must be identical to client 0's (elastic/beats#50376)", idx, j)
			}
		}
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
