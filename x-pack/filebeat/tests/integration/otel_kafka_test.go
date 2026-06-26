// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/beats/v7/x-pack/otel/oteltestcol"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/sarama"
)

// kafkaBroker is the Kafka OUTSIDE listener, which advertises localhost:9094
const kafkaBroker = "localhost:9094"

func TestFilebeatOTelKafkaE2E(t *testing.T) {
	numEvents := 1

	tmpdir := t.TempDir()
	fbTopic := fmt.Sprintf("test-fb-kafka-%s", uuid.Must(uuid.NewV4()).String())
	otelTopic := fmt.Sprintf("test-otel-kafka-%s", uuid.Must(uuid.NewV4()).String())

	logFilePath := filepath.Join(tmpdir, "kafka_e2e.log")
	writeEventsToLogFile(t, logFilePath, numEvents)

	otelCfg := fmt.Sprintf(`receivers:
  filebeatreceiver:
    include_metadata: true
    filebeat:
      inputs:
        - type: filestream
          id: filestream-input-id
          enabled: true
          paths:
            - %s
          prospector.scanner.fingerprint.enabled: false
          file_identity.native: ~
    processors:
      - add_host_metadata: ~
      - add_cloud_metadata: ~
      - add_docker_metadata: ~
      - add_kubernetes_metadata: ~
    queue.mem.flush.timeout: 0s
    setup.template.enabled: false
    path.home: %s
exporters:
  kafka:
    brokers:
      - %s
    logs:
      topic: %s
      encoding: raw
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      exporters:
        - kafka
  telemetry:
    metrics:
      level: none
`, logFilePath, tmpdir, kafkaBroker, otelTopic)

	oteltestcol.New(t, otelCfg)

	fbCfg := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: filestream-input-id
    enabled: true
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
    paths:
      - %s
output:
  kafka:
    hosts:
      - %s
    topic: %s
queue.mem.flush.timeout: 0s
setup.template.enabled: false
processors:
    - add_host_metadata: ~
    - add_cloud_metadata: ~
    - add_docker_metadata: ~
    - add_kubernetes_metadata: ~
`, logFilePath, kafkaBroker, fbTopic)

	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")
	filebeat.WriteConfigFile(fbCfg)
	filebeat.Start()
	defer filebeat.Stop()

	otelMsg := consumeKafkaTopic(t, otelTopic)
	fbMsg := consumeKafkaTopic(t, fbTopic)

	t.Logf("otel kafka message: %s", string(otelMsg))
	t.Logf("filebeat kafka message: %s", string(fbMsg))

	var otelBody, fbBody mapstr.M
	require.NoError(t, json.Unmarshal(otelMsg, &otelBody), "OTel kafka message is not valid JSON")
	require.NoError(t, json.Unmarshal(fbMsg, &fbBody), "filebeat kafka message is not valid JSON")

	assert.NotEmpty(t, otelBody["@metadata"], "expected @metadata to be present in OTel kafka message")

	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"log.file.inode",
		"log.file.device_id",
	}

	oteltest.AssertMapsEqual(t, fbBody, otelBody, ignoredFields, "expected documents to be equal")

	assert.Equal(t, "filebeat", otelBody.Flatten()["agent.type"], "expected agent.type to be 'filebeat' in otel doc")
	assert.Equal(t, "filebeat", fbBody.Flatten()["agent.type"], "expected agent.type to be 'filebeat' in filebeat doc")
}

// consumeKafkaTopic waits for a topic to appear and returns the first message received.
func consumeKafkaTopic(t *testing.T, topic string) []byte {
	t.Helper()

	cfg := sarama.NewConfig()
	cfg.Consumer.Return.Errors = true

	t.Cleanup(func() {
		admin, err := sarama.NewClusterAdmin([]string{kafkaBroker}, cfg)
		if err != nil {
			t.Logf("failed to create cluster admin for topic cleanup: %v", err)
			return
		}
		defer admin.Close()
		if err := admin.DeleteTopic(topic); err != nil {
			t.Logf("failed to delete topic %q: %v", topic, err)
		}
	})

	consumer, err := sarama.NewConsumer([]string{kafkaBroker}, cfg)
	require.NoError(t, err, "failed to create Kafka consumer")
	t.Cleanup(func() { _ = consumer.Close() })

	msgs := make(chan *sarama.ConsumerMessage, 10)
	require.Eventually(t, func() bool {
		partitions, err := consumer.Partitions(topic)
		if err != nil || len(partitions) == 0 {
			return false
		}
		for _, partition := range partitions {
			pc, err := consumer.ConsumePartition(topic, partition, sarama.OffsetOldest)
			if err != nil {
				continue
			}
			t.Cleanup(func() { _ = pc.Close() })
			go func(pc sarama.PartitionConsumer) {
				for msg := range pc.Messages() {
					msgs <- msg
				}
			}(pc)
		}
		return true
	}, 30*time.Second, 500*time.Millisecond, "topic %q did not appear within 30s", topic)

	var received []byte
	require.Eventually(t, func() bool {
		select {
		case msg := <-msgs:
			received = msg.Value
			return len(received) > 0
		default:
			return false
		}
	}, 30*time.Second, 500*time.Millisecond, "no message received from Kafka topic %q", topic)

	return received
}
