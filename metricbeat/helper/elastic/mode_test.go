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

package elastic

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"

	"github.com/stretchr/testify/require"
)

func TestModeUnpack(t *testing.T) {
	tests := map[string]struct {
		modeStr       string
		expectedMode  Mode
		isErrExpected bool
	}{
		"default": {
			modeStr:      "default",
			expectedMode: ModeDefault,
		},
		"stack-monitoring": {
			modeStr:      "stack-monitoring",
			expectedMode: ModeStackMonitoring,
		},
		"invalid": {
			modeStr:       "foobar",
			isErrExpected: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := common.MustNewConfigFrom(map[string]string{
				"mode": test.modeStr,
			})

			var config ModeConfig
			err := cfg.Unpack(&config)
			if test.isErrExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedMode, config.Mode)
			}
		})
	}
}

func TestGetMode(t *testing.T) {
	tests := map[string]struct {
		config       ModeConfig
		expectedMode Mode
	}{
		"xpack_disabled_mode_default": {
			ModeConfig{
				XPackEnabled: false,
				Mode:         ModeDefault,
			},
			ModeDefault,
		},
		"xpack_enabled_mode_default": {
			ModeConfig{
				XPackEnabled: true,
				Mode:         ModeDefault,
			},
			ModeStackMonitoring,
		},
		"xpack_disabled_mode_stack_monitoring": {
			ModeConfig{
				XPackEnabled: false,
				Mode:         ModeStackMonitoring,
			},
			ModeStackMonitoring,
		},
		"xpack_enabled_mode_stack_monitoring": {
			ModeConfig{
				XPackEnabled: true,
				Mode:         ModeStackMonitoring,
			},
			ModeStackMonitoring,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mode := test.config.GetMode()
			require.Equal(t, test.expectedMode, mode)
		})
	}
}
