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
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/otel/oteltestcol"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/sarama"
)

// kafkaBroker is the Kafka OUTSIDE listener, which advertises localhost:9094
const kafkaBroker = "localhost:9094"

func TestFilebeatOTelKafkaExporter(t *testing.T) {
	topic := fmt.Sprintf("test-otel-kafka-%s", uuid.Must(uuid.NewV4()).String())

	tmpdir := t.TempDir()
	logFilePath := filepath.Join(tmpdir, "kafka_test.log")
	writeEventsToLogFile(t, logFilePath, 1)

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
      - add_formatted_index:
          index: logs-test-default
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
`, logFilePath, tmpdir, kafkaBroker, topic)

	oteltestcol.New(t, otelCfg)

	received := consumeKafkaTopic(t, topic)

	t.Logf("received Kafka message: %s", string(received))

	var body mapstr.M
	require.NoError(t, json.Unmarshal(received, &body), "Kafka message is not valid JSON")
	got := body.Flatten()

	// Check non-deterministic fields are present.
	agentVersion, _ := got.GetValue("agent.version")
	require.NotEmpty(t, agentVersion, "expected agent.version to be set")
	agentID, _ := got.GetValue("agent.id")
	require.NotEmpty(t, agentID, "expected agent.id to be set")
	agentEphemeralID, _ := got.GetValue("agent.ephemeral_id")
	require.NotEmpty(t, agentEphemeralID, "expected agent.ephemeral_id to be set")
	hostName, _ := got.GetValue("host.name")
	require.NotEmpty(t, hostName, "expected host.name to be set")
	timestamp, _ := got.GetValue("@timestamp")
	require.NotEmpty(t, timestamp, "expected @timestamp to be set")

	// Remove non-deterministic fields before comparison.
	_ = got.Delete("@timestamp")
	_ = got.Delete("agent.id")
	_ = got.Delete("agent.ephemeral_id")
	_ = got.Delete("agent.name")
	_ = got.Delete("agent.version")
	_ = got.Delete("host.name")
	_ = got.Delete("log.file.device_id")
	_ = got.Delete("log.file.inode")

	want := mapstr.M{
		"message":             "Line 0",
		"agent.type":          "filebeat",
		"input.type":          "filestream",
		"ecs.version":         "8.0.0",
		"log.offset":          float64(0),
		"log.file.path":       logFilePath,
		"@metadata.raw_index": "logs-test-default",
		"@metadata.beat":      "filebeat",
		"@metadata.type":      "_doc",
		"@metadata.version":   agentVersion,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("log event fields mismatch (-want +got):\n%s", diff)
	}
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
