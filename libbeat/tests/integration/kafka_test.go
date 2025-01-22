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
	"fmt"
	"testing"
	"time"

	"github.com/elastic/sarama"
)

var (
	// https://github.com/elastic/sarama/blob/c7eabfcee7e5bcd7d0071f0ece4d6bec8c33928a/config_test.go#L14-L17
	// The version of MockBroker used when this test was written only supports the lowest protocol version by default.
	// Version incompatibilities will result in message decoding errors between the mock and the beat.
	kafkaVersion = sarama.MinVersion
	kafkaTopic   = "test_topic"
	kafkaCfg     = `
mockbeat:
logging:
  level: debug
  selectors:
    - publisher_pipeline_output
    - kafka
queue.mem:
  events: 4096
  flush.timeout: 0s
output.kafka:
  topic: %s
  version: %s
  hosts:
    - %s
  backoff:
    init: 0.1s
    max: 0.2s
`
)

// TestKafkaOutputCanConnectAndPublish ensures the beat Kafka output can successfully produce messages to Kafka.
// Regression test for https://github.com/elastic/beats/issues/41823 where the Kafka output would
// panic on the first Publish because it's Connect method was no longer called.
func TestKafkaOutputCanConnectAndPublish(t *testing.T) {
	// Create a Mock Kafka broker that will listen on localhost on a random unallocated port.
	// The reference configuration was taken from https://github.com/elastic/sarama/blob/c7eabfcee7e5bcd7d0071f0ece4d6bec8c33928a/async_producer_test.go#L141.
	leader := sarama.NewMockBroker(t, 1)
	defer leader.Close()

	// The mock broker must respond to a single metadata request.
	metadataResponse := new(sarama.MetadataResponse)
	metadataResponse.AddBroker(leader.Addr(), leader.BrokerID())
	metadataResponse.AddTopicPartition(kafkaTopic, 0, leader.BrokerID(), nil, nil, nil, sarama.ErrNoError)
	leader.Returns(metadataResponse)

	// The mock broker must return a single produce response. If no produce request is received, the test will fail.
	// This guarantees that mockbeat successfully produced a message to Kafka and connectivity is established.
	prodSuccess := new(sarama.ProduceResponse)
	prodSuccess.AddTopicPartition(kafkaTopic, 0, sarama.ErrNoError)
	leader.Returns(prodSuccess)

	// Start mockbeat with the appropriate configuration.
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(kafkaCfg, kafkaTopic, kafkaVersion, leader.Addr()))
	mockbeat.Start()

	// Wait for mockbeat to log that it successfully published a batch to Kafka.
	// This ensures that mockbeat received the expected produce response configured above.
	mockbeat.WaitForLogs(
		`finished kafka batch`,
		10*time.Second,
		"did not find finished batch log")
}

func TestAuthorisationErrors(t *testing.T) {
	leader := sarama.NewMockBroker(t, 1)
	defer leader.Close()

	// The mock broker must respond to a single metadata request.
	metadataResponse := new(sarama.MetadataResponse)
	metadataResponse.AddBroker(leader.Addr(), leader.BrokerID())
	metadataResponse.AddTopicPartition(kafkaTopic, 0, leader.BrokerID(), nil, nil, nil, sarama.ErrNoError)
	leader.Returns(metadataResponse)

	authErrors := []sarama.KError{
		sarama.ErrTopicAuthorizationFailed,
		sarama.ErrGroupAuthorizationFailed,
		sarama.ErrClusterAuthorizationFailed,
	}

	// The mock broker must return one produce response per error we want
	// to test. If less calls are made, the test will fail
	for _, err := range authErrors {
		producerResponse := new(sarama.ProduceResponse)
		producerResponse.AddTopicPartition(kafkaTopic, 0, err)
		leader.Returns(producerResponse)
	}

	// Start mockbeat with the appropriate configuration.
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(kafkaCfg, kafkaTopic, kafkaVersion, leader.Addr()))
	mockbeat.Start()

	// Wait for mockbeat to log each of the errors.
	for _, err := range authErrors {
		t.Log("waiting for:", err)
		mockbeat.WaitForLogs(
			fmt.Sprintf("Kafka (topic=test_topic): authorisation error: %s", err),
			10*time.Second,
			"did not find error log: %s", err)
	}
}
