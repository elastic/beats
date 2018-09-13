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

package mtest

import (
	"io"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	saramacluster "github.com/bsm/sarama-cluster"
)

func GenerateKafkaData(t *testing.T, topic string, host string) {
	t.Logf("Send Kafka Event to topic: %v", topic)

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	// Retry for 10 seconds
	config.Producer.Retry.Max = 20
	config.Producer.Retry.Backoff = 500 * time.Millisecond
	config.Metadata.Retry.Max = 20
	config.Metadata.Retry.Backoff = 500 * time.Millisecond
	client, err := sarama.NewClient([]string{host}, config)
	if err != nil {
		t.Errorf("%s", err)
		t.FailNow()
	}

	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		t.Error(err)
	}
	defer producer.Close()

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder("Hello World"),
	}

	_, _, err = producer.SendMessage(msg)
	if err != nil {
		t.Errorf("failed to send message: %s\n", err)
	}

	err = client.RefreshMetadata(topic)
	if err != nil {
		t.Errorf("failed to refresh metadata for topic '%s': %s\n", topic, err)
	}
}

func StartConsumer(t *testing.T, topic, host string) (io.Closer, error) {
	brokers := []string{host}
	topics := []string{topic}

	return saramacluster.NewConsumer(brokers, "test-group", topics, nil)
}
