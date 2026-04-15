// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/sarama"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/otel/oteltestcol"
)

const otelKafkaTopic = "test-otel-kafka"

// kafkaBroker is the Kafka OUTSIDE listener, which advertises localhost:9094
// so it is reachable from the test host. The INSIDE listener on port 9092
// advertises kafka:9092 (docker-internal hostname) and cannot be used here.
const kafkaBroker = "localhost:9094"

func TestFilebeatOTelKafkaExporter(t *testing.T) {
	tmpdir := t.TempDir()
	logFilePath := filepath.Join(tmpdir, "kafka_test.log")
	writeEventsToLogFile(t, logFilePath, 1)

	otelCfg := fmt.Sprintf(`receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          id: filestream-input-id
          enabled: true
          paths:
            - %s
          prospector.scanner.fingerprint.enabled: false
          file_identity.native: ~
    queue.mem.flush.timeout: 0s
    setup.template.enabled: false
    path.home: %s
exporters:
  kafka:
    brokers:
      - %s
    logs:
      topic: %s
      encoding: otlp_json
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
`, logFilePath, tmpdir, kafkaBroker, otelKafkaTopic)

	oteltestcol.New(t, otelCfg)

	cfg := sarama.NewConfig()
	cfg.Consumer.Return.Errors = true

	// Create the consumer once; retry until Kafka is reachable.
	consumer, err := sarama.NewConsumer([]string{kafkaBroker}, cfg)
	require.NoError(t, err, "failed to create Kafka consumer")
	t.Cleanup(func() { _ = consumer.Close() })

	// Wait for the topic to appear (the exporter creates it on first produce).
	msgs := make(chan *sarama.ConsumerMessage, 10)
	require.Eventually(t, func() bool {
		partitions, err := consumer.Partitions(otelKafkaTopic)
		if err != nil || len(partitions) == 0 {
			return false
		}
		// Spin up one goroutine per partition forwarding into msgs.
		for _, partition := range partitions {
			pc, err := consumer.ConsumePartition(otelKafkaTopic, partition, sarama.OffsetOldest)
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
	}, 30*time.Second, 500*time.Millisecond, "topic %q did not appear within 30s", otelKafkaTopic)

	// Poll until at least one message arrives.
	var received []byte
	require.Eventually(t, func() bool {
		select {
		case msg := <-msgs:
			received = msg.Value
			return len(received) > 0
		default:
			return false
		}
	}, 30*time.Second, 500*time.Millisecond, "no message received from Kafka topic %q", otelKafkaTopic)

	t.Logf("received Kafka message: %s", string(received))
	require.NotEmpty(t, received, "expected a non-empty message from Kafka topic %q", otelKafkaTopic)
}
