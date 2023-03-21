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
		r.cfgs = append(r.cfgs, client.(*clientMock).cfg)
	}
	return &struct {
		cfgfile.Runner
	}{}, nil
}

func (runnerFactoryMock) CheckConfig(config *conf.C) error {
	return nil
}

// Assert runs various checks for the clients created by the wrapped pipeline connector
// We check that the processing configuration does not reference the same addresses as before,
// re-using some parts of the processing configuration will result in various issues, such as:
// * closing processors multiple times
// * using closed processors
// * modifiying an object shared by multiple pipeline clients
func (r runnerFactoryMock) Assert(t *testing.T) {
	t.Helper()

	// we need to make sure `Assert` is called after `Create`
	require.Len(t, r.cfgs, r.clientCount)

	t.Run("new processing configuration each time", func(t *testing.T) {
		for i, c1 := range r.cfgs {
			for j, c2 := range r.cfgs {
				if i == j {
					continue
				}

				require.NotSamef(t, c1.Processing, c2.Processing, "processing configuration cannot be re-used")
				require.NotSamef(t, c1.Processing.Meta, c2.Processing.Meta, "`Processing.Meta` cannot be re-used")
				require.NotSamef(t, c1.Processing.Fields, c2.Processing.Fields, "`Processing.Fields` cannot be re-used")
				require.NotSamef(t, c1.Processing.Processor, c2.Processing.Processor, "`Processing.Processor` cannot be re-used")
			}
		}
	})

	t.Run("new processors each time", func(t *testing.T) {
		var processors []beat.Processor
		for _, c := range r.cfgs {
			processors = append(processors, c.Processing.Processor.All()...)
		}

		require.NotEmptyf(t, processors, "for this test the list of processors cannot be empty")

		for i, p1 := range processors {
			for j, p2 := range processors {
				if i == j {
					continue
				}

				require.NotSamef(t, p1, p2, "processors must not be re-used")
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
