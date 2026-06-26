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

package testutil

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/sarama"
)

const (
	kafkaDefaultHost     = "localhost"
	kafkaDefaultPort     = "9094"
	kafkaDefaultSASLPort = "9093"
)

func RecordHeader(key, value string) sarama.RecordHeader {
	return sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(value),
	}
}

func WriteToKafkaTopic(
	t *testing.T, topic string, message string,
	headers []sarama.RecordHeader,
) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Return.Successes = true
	config.Producer.Partitioner = sarama.NewHashPartitioner
	config.Version = sarama.V1_0_0_0
	config.Producer.Retry.Max = 10
	config.Producer.Retry.Backoff = 100 * time.Millisecond

	hosts := []string{GetTestKafkaHost()}

	// Retry producer creation to handle transient connection issues
	var producer sarama.SyncProducer
	var err error
	require.EventuallyWithTf(t, func(ct *assert.CollectT) {
		producer, err = sarama.NewSyncProducer(hosts, config)
		require.NoError(ct, err)
		require.NotNil(ct, producer)
	}, 30*time.Second, 1*time.Second, "failed to create producer: %v", err)

	defer func() {
		if err := producer.Close(); err != nil {
			require.NoError(t, err)
		}
	}()

	msg := &sarama.ProducerMessage{
		Topic:   topic,
		Value:   sarama.StringEncoder(message),
		Headers: headers,
	}

	_, _, err = producer.SendMessage(msg)
	require.NoError(t, err)
}

func GetTestKafkaHost() string {
	return fmt.Sprintf("%v:%v",
		getenv("KAFKA_HOST", kafkaDefaultHost),
		getenv("KAFKA_PORT", kafkaDefaultPort),
	)
}

func GetTestSASLKafkaHost() string {
	return fmt.Sprintf("%v:%v",
		getenv("KAFKA_HOST", kafkaDefaultHost),
		getenv("KAFKA_SASL_PORT", kafkaDefaultSASLPort),
	)
}

func getenv(name, defaultValue string) string {
	return strDefault(os.Getenv(name), defaultValue)
}

func strDefault(a, defaults string) string {
	if a == "" {
		return defaults
	}
	return a
}

func EnsureKafkaTopicReadyForWrites(t *testing.T, topic string) {
	config := sarama.NewConfig()
	config.Version = sarama.V1_0_0_0
	hosts := []string{GetTestKafkaHost()}

	admin, err := sarama.NewClusterAdmin(hosts, config)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, admin.Close())
	})

	topicDetail := &sarama.TopicDetail{
		NumPartitions:     3,
		ReplicationFactor: 1,
	}
	require.EventuallyWithTf(t, func(ct *assert.CollectT) {
		err = admin.CreateTopic(topic, topicDetail, false)
		if err != nil && !errors.Is(err, sarama.ErrTopicAlreadyExists) {
			require.NoError(ct, err)
		}
	}, 30*time.Second, 200*time.Millisecond, "failed to create topic %s", topic)

	client, err := sarama.NewClient(hosts, config)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})

	require.EventuallyWithTf(t, func(ct *assert.CollectT) {
		require.NoError(ct, client.RefreshMetadata(topic))

		partitions, err := client.Partitions(topic)
		require.NoError(ct, err)
		require.NotEmpty(ct, partitions)

		for _, partition := range partitions {
			leader, err := client.Leader(topic, partition)
			require.NoError(ct, err)
			require.NotNil(ct, leader)
		}
	}, 30*time.Second, 200*time.Millisecond, "topic %s is not ready for writes", topic)
}
