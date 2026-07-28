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
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
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
			pathsFunc: func(string) *paths.Path { return paths.New() },
			wantStatePath: func(tempDir string) string {
				return filepath.Join(tempDir, "state.dat")
			},
		},
		{
			name: "falls back to beat data path",
			cfg: func(string) string {
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

			beatPaths := tt.pathsFunc(tempDir)

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

func TestSuccessNetLoadBalanceFalse(t *testing.T) {
	const (
		batchSize = 10
		retry     = 3
	)

	logger := logptest.NewTestingLogger(t, "test-successnet")
	beatPaths := paths.New()
	var queueCfg config.Namespace

	t.Run("worker_one_single_backend", func(t *testing.T) {
		backend := &stubNetworkClient{id: 0}
		group, err := SuccessNet(queueCfg, false, batchSize, retry, nil, logger, beatPaths, 1, []NetworkClient{backend})
		require.NoError(t, err)
		require.Len(t, group.Clients, 1)
		assert.Same(t, backend, group.Clients[0])
	})

	t.Run("worker_one_multiple_backends_one_failover", func(t *testing.T) {
		netclients := []NetworkClient{
			&stubNetworkClient{id: 0},
			&stubNetworkClient{id: 1},
			&stubNetworkClient{id: 2},
		}
		group, err := SuccessNet(queueCfg, false, batchSize, retry, nil, logger, beatPaths, 1, netclients)
		require.NoError(t, err)
		require.Len(t, group.Clients, 1)

		failover, ok := group.Clients[0].(*failoverClient)
		require.True(t, ok)
		require.Len(t, failover.clients, 3)
		assert.Equal(t, netclients, failover.clients)
	})

	t.Run("worker_two_columns_per_host", func(t *testing.T) {
		netclients := []NetworkClient{
			&stubNetworkClient{id: 0},
			&stubNetworkClient{id: 0},
			&stubNetworkClient{id: 1},
			&stubNetworkClient{id: 1},
		}
		group, err := SuccessNet(queueCfg, false, batchSize, retry, nil, logger, beatPaths, 2, netclients)
		require.NoError(t, err)
		require.Len(t, group.Clients, 2)

		col0, ok := group.Clients[0].(*failoverClient)
		require.True(t, ok)
		assert.Equal(t, []NetworkClient{netclients[0], netclients[2]}, col0.clients)

		col1, ok := group.Clients[1].(*failoverClient)
		require.True(t, ok)
		assert.Equal(t, []NetworkClient{netclients[1], netclients[3]}, col1.clients)
	})

	t.Run("worker_mismatch", func(t *testing.T) {
		netclients := []NetworkClient{
			&stubNetworkClient{id: 0},
			&stubNetworkClient{id: 1},
			&stubNetworkClient{id: 2},
		}
		_, err := SuccessNet(queueCfg, false, batchSize, retry, nil, logger, beatPaths, 2, netclients)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match host list")
	})

	t.Run("worker_defaults_to_one", func(t *testing.T) {
		backend := &stubNetworkClient{id: 0}
		group, err := SuccessNet(queueCfg, false, batchSize, retry, nil, logger, beatPaths, 0, []NetworkClient{backend})
		require.NoError(t, err)
		require.Len(t, group.Clients, 1)
		assert.Same(t, backend, group.Clients[0])
	})
}

type stubNetworkClient struct {
	id int
}

func (c *stubNetworkClient) Close() error { return nil }

func (c *stubNetworkClient) Connect(_ context.Context) error { return nil }

func (c *stubNetworkClient) Publish(_ context.Context, batch publisher.Batch) error {
	batch.ACK()
	return nil
}

func (c *stubNetworkClient) String() string { return "stub" }
