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

package outputs

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

func TestDiskQueueUnderAgent(t *testing.T) {
	const (
		batchSize = 10
		retry     = 3
	)

	tests := []struct {
		name          string
		cfg           func(tempDir string) string
		pathsFunc     func(tempDir string) *paths.Path
		wantStatePath func(tempDir string) string
	}{
		{
			name: "explicit path in config",
			cfg: func(tempDir string) string {
				return fmt.Sprintf(`
                    disk:
                        max_size: 100MB
                        path: %s
                `, tempDir)
			},
			wantStatePath: func(tempDir string) string {
				return filepath.Join(tempDir, "state.dat")
			},
		},
		{
			name: "falls back to beat data path",
			cfg: func(_ string) string {
				return `
                    disk:
                        max_size: 100MB
                `
			},
			pathsFunc: func(tempDir string) *paths.Path {
				return &paths.Path{Data: tempDir}
			},
			wantStatePath: func(tempDir string) string {
				return filepath.Join(tempDir, "diskqueue", "state.dat")
			},
		},
		{
			name: "explicit path takes precedence over data path",
			cfg: func(tempDir string) string {
				return fmt.Sprintf(`
                    disk:
                        max_size: 100MB
                        path: %s
                `, filepath.Join(tempDir, "explicit"))
			},
			pathsFunc: func(tempDir string) *paths.Path {
				return &paths.Path{Data: filepath.Join(tempDir, "data")}
			},
			wantStatePath: func(tempDir string) string {
				return filepath.Join(tempDir, "explicit", "state.dat")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previousUnderAgent := management.UnderAgent()
			t.Cleanup(func() {
				management.SetUnderAgent(previousUnderAgent)
			})

			tempDir := t.TempDir()

			queueConfig := config.Namespace{}
			conf, err := config.NewConfigFrom(tt.cfg(tempDir))
			require.NoError(t, err, "error parsing queue config")
			err = queueConfig.Unpack(conf)
			require.NoError(t, err, "error unpacking queue config")

			management.SetUnderAgent(true)

			beatPaths := paths.New()
			if tt.pathsFunc != nil {
				beatPaths = tt.pathsFunc(tempDir)
			}

			successLogger, logBuf := logp.NewInMemoryLocal("test-diskqueue", zapcore.EncoderConfig{})
			group, err := Success(queueConfig, batchSize, retry, nil, successLogger, beatPaths)
			require.NoError(t, err)
			require.NotNil(t, group)
			require.NotNil(t, group.QueueFactory)

			assert.Contains(t, logBuf.String(), "unsupported and in technical preview")

			queueLogger, _ := logp.NewInMemoryLocal("test-diskqueue", zapcore.EncoderConfig{})
			q, err := group.QueueFactory(queueLogger, nil, 1, nil)
			require.NoError(t, err)
			require.NotNil(t, q)
			defer func() { require.NoError(t, q.Close(true)) }()

			assert.FileExists(t, tt.wantStatePath(tempDir))
		})
	}
}
