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

//go:build !integration
// +build !integration

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/common"
)

type validationTestCase struct {
	config WinlogbeatConfig
	errMsg string
}

func (v validationTestCase) run(t *testing.T) {
	if v.errMsg == "" {
		assert.NoError(t, v.config.Validate())
	} else {
		err := v.config.Validate()
		if err != nil {
			assert.Contains(t, err.Error(), v.errMsg)
		} else {
			t.Errorf("expected error with '%s'", v.errMsg)
		}
	}
}

func TestConfigValidate(t *testing.T) {
	testCases := []validationTestCase{
		// Top-level config
		{
			WinlogbeatConfig{
				EventLogs: []*common.Config{
					newConfig(map[string]interface{}{
						"Name": "App",
					}),
				},
			},
			"", // No Error
		},
		{
			WinlogbeatConfig{},
			"1 error: at least one event log must be configured as part of " +
				"event_logs",
		},
	}

	for _, test := range testCases {
		test.run(t)
	}
}

func newConfig(from map[string]interface{}) *common.Config {
	cfg, err := common.NewConfigFrom(from)
	if err != nil {
		panic(err)
	}
	return cfg
}
