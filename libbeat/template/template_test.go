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

// +build !integration

package template

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestNumberOfRoutingShards(t *testing.T) {

	beatVersion := "6.1.0"
	beatName := "testbeat"
	config := TemplateConfig{}

	// Test it exists in 6.1
	ver := common.MustNewVersion("6.1.0")
	template, err := New(beatVersion, beatName, *ver, config)
	assert.NoError(t, err)

	data := template.Generate(nil, nil)
	shards, err := data.GetValue("settings.index.number_of_routing_shards")
	assert.NoError(t, err)

	assert.Equal(t, 30, shards.(int))

	// Test it does not exist in 6.0
	ver = common.MustNewVersion("6.0.0")
	template, err = New(beatVersion, beatName, *ver, config)
	assert.NoError(t, err)

	data = template.Generate(nil, nil)
	shards, err = data.GetValue("settings.index.number_of_routing_shards")
	assert.Error(t, err)
	assert.Equal(t, nil, shards)
}

func TestNumberOfRoutingShardsOverwrite(t *testing.T) {

	beatVersion := "6.1.0"
	beatName := "testbeat"
	config := TemplateConfig{
		Settings: TemplateSettings{
			Index: map[string]interface{}{"number_of_routing_shards": 5},
		},
	}

	// Test it exists in 6.1
	ver := common.MustNewVersion("6.1.0")
	template, err := New(beatVersion, beatName, *ver, config)
	assert.NoError(t, err)

	data := template.Generate(nil, nil)
	shards, err := data.GetValue("settings.index.number_of_routing_shards")
	assert.NoError(t, err)

	assert.Equal(t, 5, shards.(int))
}
