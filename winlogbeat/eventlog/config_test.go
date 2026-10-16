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

//go:build windows

package eventlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestReadConfig_StartPosition(t *testing.T) {
	tests := []struct {
		name    string
		cfg     map[string]any
		want    StartPosition
		wantErr string
	}{
		{
			name: "default is beginning",
			cfg:  map[string]any{"name": "Application"},
			want: StartAtBeginning,
		},
		{
			name: "end",
			cfg:  map[string]any{"name": "Application", "start_position": "end"},
			want: StartAtEnd,
		},
		{
			name:    "invalid value",
			cfg:     map[string]any{"name": "Application", "start_position": "oldest"},
			wantErr: "invalid start_position: oldest",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := defaultConfig()
			err := readConfig(conf.MustNewConfigFrom(tc.cfg), &c)
			if tc.wantErr != "" {
				require.Error(t, err, "expected config to be rejected")
				assert.Contains(t, err.Error(), tc.wantErr, "unexpected error message")
				return
			}
			require.NoError(t, err, "config should be accepted")
			assert.Equal(t, tc.want, c.StartPosition, "unexpected start_position")
		})
	}
}
