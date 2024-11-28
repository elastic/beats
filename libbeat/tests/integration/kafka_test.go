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
	"testing"
	"time"
)

var kafkaCfg = `
mockbeat:
logging:
  level: debug
  selectors:
    - publisher_pipeline_output
    - kafka
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.kafka:
  topic: test
  hosts:
    - "localhost:9092"
  backoff:
    init: 0.1s
    max: 0.2s
`

// Regression test for https://github.com/elastic/beats/issues/41823
// The Kafka output would panic on the first Publish because it's Connect method was no longer called.
func TestKafkaOutputCanConnect(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(kafkaCfg)

	mockbeat.Start()

	// 3. Wait for connection error logs
	mockbeat.WaitForLogs(
		`Connection to kafka(localhost:9092) established`,
		15*time.Second,
		"did not find connection establishment log")

	mockbeat.WaitForLogs(
		"Kafka publish failed with: kafka: client has run out of available brokers to talk to (Is your cluster reachable?",
		5*time.Second,
		"did not find message from Kafka producer after connecting")
}
