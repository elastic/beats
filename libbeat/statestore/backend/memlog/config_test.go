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

package memlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid default (field absent)",
			yaml: "",
		},
		{
			name: "valid 10MB",
			yaml: "checkpoint_size: 10485760",
		},
		{
			name: "valid 50MB",
			yaml: "checkpoint_size: 52428800",
		},
		{
			name:    "too small 1 byte",
			yaml:    "checkpoint_size: 1",
			wantErr: true,
		},
		{
			name:    "too small 9MB",
			yaml:    "checkpoint_size: 9437184",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()

			raw, err := conf.NewConfigWithYAML([]byte(tt.yaml), "test")
			require.NoError(t, err, "failed to parse test YAML")

			err = raw.Unpack(&cfg)
			if tt.wantErr {
				require.Error(t, err, "expected validation error")
				assert.Contains(t, err.Error(), "requires value >= 10485760",
					"error message should mention the minimum")
			} else {
				assert.NoError(t, err, "expected valid config")
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, uint64(defaultCheckpointSize), cfg.CheckpointSize,
		"default checkpoint size should be 10 MB")
}
