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

package inputtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Outlet is an empty outlet for testing.
type Outlet struct{}

func (o Outlet) OnEvent(event beat.Event) bool { return true }
func (o Outlet) Close() error                  { return nil }
func (o Outlet) Done() <-chan struct{}         { return nil }

// Connector is a connector to a test empty outlet.
var Connector = channel.ConnectorFunc(
	func(_ *common.Config, _ beat.ClientConfig) (channel.Outleter, error) {
		return Outlet{}, nil
	},
)

// AssertNotStartedInputCanBeDone checks that the context of an input can be
// done before starting the input, and it doesn't leak goroutines. This is
// important to confirm that leaks don't happen with CheckConfig.
func AssertNotStartedInputCanBeDone(t *testing.T, factory input.Factory, configMap *mapstr.M) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	config, err := common.NewConfigFrom(configMap)
	require.NoError(t, err)

	context := input.Context{
		Done: make(chan struct{}),
	}

	_, err = factory(config, Connector, context)
	assert.NoError(t, err)

	close(context.Done)
}
