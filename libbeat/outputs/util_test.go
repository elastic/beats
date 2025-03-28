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
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestDiskQueueUnderAgent(t *testing.T) {

	type args struct {
		cfg            string
		batchSize      int
		retry          int
		encoderFactory queue.EncoderFactory
		clients        []Client
	}
	tests := []struct {
		name    string
		args    args
		want    Group
		wantErr bool
	}{
		{
			name: "Happy path",
			args: args{
				cfg: `
                    disk:
                        max_size: 100MB
                        path: %s
                `,
				clients:   []Client{},
				batchSize: 10,
				retry:     3,
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
			conf, err := config.NewConfigFrom(fmt.Sprintf(tt.args.cfg, tempDir))
			require.NoError(t, err, "error parsing queue config")
			err = queueConfig.Unpack(conf)
			require.NoError(t, err, "error unpacking queue config")

			management.SetUnderAgent(true)

			actualGroup, err := Success(queueConfig, tt.args.batchSize, tt.args.retry, tt.args.encoderFactory, tt.args.clients...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Success() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				// if an error was expected, we need no more assertions: return
				return
			}

			require.NotNil(t, actualGroup)
			require.NotNil(t, actualGroup.QueueFactory)

			testlogger, _ := logp.NewInMemoryLocal("test-diskqueue", zapcore.EncoderConfig{})
			actualQueue, err := actualGroup.QueueFactory(testlogger, nil, 1, nil)
			require.NoError(t, err)
			require.NotNil(t, actualQueue)
			// assert that the file exists in the path we specified
			assert.FileExists(t, filepath.Join(tempDir, "state.dat"))
		})
	}
}
