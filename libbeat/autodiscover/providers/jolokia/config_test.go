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

package jolokia

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
)

func TestInterfaceConfigsUnpack(t *testing.T) {
	cases := []struct {
		Description string
		Config      map[string]interface{}
		Valid       bool
	}{
		{
			Description: "interface without name",
			Config:      map[string]interface{}{},
			Valid:       false,
		},
		{
			Description: "interface with empty name",
			Config: map[string]interface{}{
				"name": "",
			},
			Valid: false,
		},
		{
			Description: "interface with zero interval",
			Config: map[string]interface{}{
				"name":     "br0",
				"interval": "0s",
			},
			Valid: false,
		},
		{
			Description: "valid interface",
			Config: map[string]interface{}{
				"name":     "br0",
				"interval": "100s",
			},
			Valid: true,
		},
	}

	for _, c := range cases {
		var ic InterfaceConfig
		config, err := common.NewConfigFrom(c.Config)
		if !assert.NoError(t, err, c.Description) {
			continue
		}
		err = config.Unpack(&ic)
		if c.Valid {
			assert.NoError(t, err, c.Description)
		} else {
			assert.Error(t, err, c.Description)
		}
	}
}

func TestDefaultConfigUnpack(t *testing.T) {
	rawConfig, err := common.NewConfigFrom(map[string]interface{}{})
	assert.NoError(t, err)
	config := defaultConfig()
	err = rawConfig.Unpack(&config)
	assert.NoError(t, err)
	assert.NotEmpty(t, config.Interfaces)
	assert.NotEmpty(t, config.Interfaces[0].Name)
}

func TestConfigUnpackEmptyInterfaces(t *testing.T) {
	rawConfig, err := common.NewConfigFrom(map[string]interface{}{
		"interfaces": []interface{}{},
	})
	assert.NoError(t, err)
	config := defaultConfig()
	err = rawConfig.Unpack(&config)
	assert.Error(t, err)
}
