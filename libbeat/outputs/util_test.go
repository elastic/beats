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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
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
		name           string
		cfg            string
		encoderFactory queue.EncoderFactory
		pathsFunc      func(string) *paths.Path
		needQueueDir   bool
	}{
		{
			name: "Happy path",
			cfg: `
                    disk:
                        max_size: 100MB
                        path: %s
                `,
		},
		{
			name: "Use data paths",
			cfg: `
                    disk:
                        max_size: 100MB
                `,
			pathsFunc: func(tempDir string) *paths.Path {
				return &paths.Path{
					Data: tempDir,
				}
			},
			needQueueDir: true,
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
			cfg := tt.cfg
			if strings.Contains(cfg, "%s") {
				cfg = fmt.Sprintf(tt.cfg, tempDir)
			}
			conf, err := config.NewConfigFrom(cfg)
			require.NoError(t, err, "error parsing queue config")
			err = queueConfig.Unpack(conf)
			require.NoError(t, err, "error unpacking queue config")

			management.SetUnderAgent(true)

			beatPaths := paths.New()
			if tt.pathsFunc != nil {
				beatPaths = tt.pathsFunc(tempDir)
			}

			actualGroup, err := Success(queueConfig, batchSize, retry, nil, logp.NewNopLogger(), beatPaths)
			require.NoError(t, err)

			require.NotNil(t, actualGroup)
			require.NotNil(t, actualGroup.QueueFactory)

			testlogger, _ := logp.NewInMemoryLocal("test-diskqueue", zapcore.EncoderConfig{})
			actualQueue, err := actualGroup.QueueFactory(testlogger, nil, 1, nil)
			require.NoError(t, err)
			require.NotNil(t, actualQueue)
			// assert that the file exists in the path we specified
			parts := []string{tempDir}
			if tt.needQueueDir {
				parts = append(parts, "diskqueue")
			}
			parts = append(parts, "state.dat")
			assert.FileExists(t, filepath.Join(parts...))
		})
	}
}
