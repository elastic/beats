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

package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
)

func TestConfigWithCustomBuilders(t *testing.T) {
	autodiscover.Registry.AddBuilder("mock", newMockBuilder)

	cfg := common.MapStr{
		"hints.enabled": false,
		"builders": []common.MapStr{
			{
				"mock": common.MapStr{},
			},
		},
	}

	config := common.MustNewConfigFrom(&cfg)
	c := defaultConfig()
	err := config.Unpack(&c)
	assert.Nil(t, err)

	cfg1 := common.MapStr{
		"hints.enabled": false,
	}
	config, err = common.NewConfigFrom(&cfg1)
	c = defaultConfig()
	err = config.Unpack(&c)
	assert.NotNil(t, err)
}

func TestConfigWithIncorrectScope(t *testing.T) {
	cfg := common.MapStr{
		"scope":         "node",
		"resource":      "service",
		"hints.enabled": true,
	}

	config := common.MustNewConfigFrom(&cfg)
	c := defaultConfig()
	err := config.Unpack(&c)
	assert.Nil(t, err)

	assert.Equal(t, "service", c.Resource)
	assert.Equal(t, "cluster", c.Scope)
}

type mockBuilder struct {
}

func newMockBuilder(_ *common.Config) (autodiscover.Builder, error) {
	return &mockBuilder{}, nil
}

func (m *mockBuilder) CreateConfig(event bus.Event) []*common.Config {
	return nil
}
