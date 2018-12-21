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

package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestConfigUnpackDefault(t *testing.T) {
	rawConfig, err := common.NewConfigFrom(map[string]interface{}{})
	assert.NoError(t, err)
	config := defaultConfig()
	err = rawConfig.Unpack(&config)
	assert.NoError(t, err)
	assert.NotEmpty(t, config.Separator)
	assert.Equal(t, "/", config.Separator)
}

func TestConfigUnpackInvalidSeparator(t *testing.T) {
	rawConfig, err := common.NewConfigFrom(map[string]interface{}{
		"separator": "#",
	})
	assert.NoError(t, err)
	config := defaultConfig()
	err = rawConfig.Unpack(&config)
	assert.Error(t, err)
}

func TestConfigUnpackValidSeparator(t *testing.T) {
	rawConfig, err := common.NewConfigFrom(map[string]interface{}{
		"separator": ".",
	})
	assert.NoError(t, err)
	config := defaultConfig()
	err = rawConfig.Unpack(&config)
	assert.NoError(t, err)
}
