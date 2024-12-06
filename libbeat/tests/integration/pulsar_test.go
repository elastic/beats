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

//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/pulsar"
)

// TestPulsarOutput tests the pulsar output with a running pulsar container.
func TestPulsarOutput(t *testing.T) {
	c, err := pulsar.Run(context.Background(), "apachepulsar/pulsar:2.10.2",
		testcontainers.WithEnv(map[string]string{"PULSAR_MEM": "-Xmx256m"}))

	require.NoError(t, err)

	timeout := 10 * time.Second
	defer func() {
		err := c.Stop(context.Background(), &timeout)
		if err != nil {
			t.Logf("error stopping pulsar container: %v", err)
		}
	}()

	addr, err := c.ContainerIP(context.Background())
	require.NoError(t, err)

	configTemplate := `
mockbeat:
logging:
  level: debug
  selectors:
    - publisher_pipeline_output
    - kafka
queue.mem:
  events: 4096
  flush.timeout: 0s
output.pulsar:
  endpoint: 'pulsar://%s:6650'
  topic: 'persistent://public/default/beats'
  producer:
    compression_type: 'lz4'
    compression_level: 'default'
    max_pending_messages: 10000
    disable_batching: false
    batch_builder_type: 'default'
    hashing_scheme: 'java_string_hash'
`
	conf := fmt.Sprintf(configTemplate, addr)

	// Start mockbeat with the appropriate configuration.
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(conf)
	mockbeat.Start()

	mockbeat.WaitForLogs(
		`finished pulsar batch`,
		10*time.Second,
		"did not find finished batch log")
}
