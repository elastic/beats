// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/input/kafka/testutil"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/beats/v7/x-pack/otel/oteltestcol"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/testing/estools"
	"github.com/elastic/sarama"
)

func TestFilebeatOTelKafkaOutputE2E(t *testing.T) {
	numEvents := 1

	tmpdir := t.TempDir()
	fbTopic := fmt.Sprintf("test-fb-kafka-%s", uuid.Must(uuid.NewV4()).String())
	otelTopic := fmt.Sprintf("test-otel-kafka-%s", uuid.Must(uuid.NewV4()).String())

	logFilePath := filepath.Join(tmpdir, "kafka_e2e.log")
	writeEventsToLogFile(t, logFilePath, numEvents)

	kafkaBroker := testutil.GetTestKafkaHost()

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
	cfg.Version = sarama.V1_0_0_0
	cfg.Consumer.Return.Errors = true

	t.Cleanup(func() {
		deleteKafkaInputTopic(t, topic)
	})

	consumer, err := sarama.NewConsumer([]string{testutil.GetTestKafkaHost()}, cfg)
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

func TestKafkaInputOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	host := integration.GetESURL(t, "http")
	user := host.User.Username()
	password, _ := host.User.Password()
	kafkaInputTestMsg := "kafka-input-otel-e2e-test-event"

	otelNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	fbNamespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")

	otelIndex := "logs-integration-" + otelNamespace
	fbIndex := "logs-filebeat-" + fbNamespace

	topic := fmt.Sprintf("test-kafka-input-%s", uuid.Must(uuid.NewV4()).String())
	otelGroupID := fmt.Sprintf("otel-kafka-%s", uuid.Must(uuid.NewV4()).String())
	fbGroupID := fmt.Sprintf("fb-kafka-%s", uuid.Must(uuid.NewV4()).String())

	testutil.EnsureKafkaTopicReadyForWrites(t, topic)
	testutil.WriteToKafkaTopic(t, topic, kafkaInputTestMsg, []sarama.RecordHeader{
		testutil.RecordHeader("X-Test-Header", "test header value"),
	})

	tempDir := t.TempDir()

	t.Cleanup(func() {
		deleteKafkaInputTopic(t, topic)
	})

	type options struct {
		ESURL    string
		Username string
		Password string
		Broker   string
		Topic    string
		GroupID  string
		PathHome string
		Index    string
	}

	kafkaFilebeatConfig := `filebeat.inputs:
- type: kafka
  id: kafka-input-e2e
  hosts:
    - {{ .Broker }}
  topics:
    - {{ .Topic }}
  group_id: {{ .GroupID }}
  wait_close: 0

output:
  elasticsearch:
    hosts:
      - {{ .ESURL }}
    username: {{ .Username }}
    password: {{ .Password }}
    index: {{ .Index}}

queue.mem.flush.timeout: 0s
setup.template.enabled: false
processors:
    - add_host_metadata: ~
    - add_cloud_metadata: ~
    - add_docker_metadata: ~
    - add_kubernetes_metadata: ~
`

	kafkaOTelConfig := otelElasticsearchExporterYAML + `receivers:
    filebeatreceiver:
        filebeat:
            inputs:
                - type: kafka
                  id: kafka-input-e2e
                  hosts:
                    - {{ .Broker }}
                  topics:
                    - {{ .Topic }}
                  group_id: {{ .GroupID }}
                  wait_close: 0
        processors:
            - add_host_metadata: ~
            - add_cloud_metadata: ~
            - add_docker_metadata: ~
            - add_kubernetes_metadata: ~
        queue.mem.flush.timeout: 0s
        setup.template.enabled: false
        management.otel.enabled: true
        path.home: {{ .PathHome }}
` + otelElasticsearchServiceYAML

	optionsValue := options{
		ESURL:    fmt.Sprintf("%s://%s", host.Scheme, host.Host),
		Username: user,
		Password: password,
		Broker:   testutil.GetTestKafkaHost(),
		Topic:    topic,
		PathHome: tempDir,
	}

	var configBuffer bytes.Buffer
	optionsValue.GroupID = otelGroupID
	optionsValue.Index = otelIndex
	require.NoError(t, template.Must(template.New("config").Parse(kafkaOTelConfig)).Execute(&configBuffer, optionsValue))

	oteltestcol.New(t, configBuffer.String())

	configBuffer.Reset()

	optionsValue.GroupID = fbGroupID
	optionsValue.Index = fbIndex
	require.NoError(t, template.Must(template.New("config").Parse(kafkaFilebeatConfig)).Execute(&configBuffer, optionsValue))

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	filebeat.WriteConfigFile(configBuffer.String())
	filebeat.Start()
	defer filebeat.Stop()

	es := integration.GetESClient(t, "http")

	t.Cleanup(func() {
		deleteDataStreamsFromES(t, es, []string{
			otelIndex,
			fbIndex,
		})
	})

	rawQuery := otelE2ERawQueryForInputTypeAndMessage("kafka", kafkaInputTestMsg)

	var filebeatDocs estools.Documents
	var otelDocs estools.Documents
	var err error

	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(t.Context(), 900*time.Millisecond)
			defer findCancel()

			otelDocs, err = estools.PerformQueryForRawQuery(findCtx, rawQuery, ".ds-"+otelIndex+"*", es)
			assert.NoError(ct, err)
			assert.GreaterOrEqual(ct, otelDocs.Hits.Total.Value, 1, "expected at least 1 otel document, got %d", otelDocs.Hits.Total.Value)

			filebeatDocs, err = estools.PerformQueryForRawQuery(findCtx, rawQuery, ".ds-"+fbIndex+"*", es)
			assert.NoError(ct, err)
			assert.GreaterOrEqual(ct, filebeatDocs.Hits.Total.Value, 1, "expected at least 1 filebeat document, got %d", filebeatDocs.Hits.Total.Value)
		},
		3*time.Minute, 1*time.Second, "expected at least 1 document for both filebeat and otel modes")

	filebeatDoc := filebeatDocs.Hits.Hits[0].Source
	otelDoc := otelDocs.Hits.Hits[0].Source
	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
	}

	oteltest.AssertMapsEqual(t, filebeatDoc, otelDoc, ignoredFields, "expected documents to be equal")
}

func deleteKafkaInputTopic(t *testing.T, topic string) {
	t.Helper()

	cfg := sarama.NewConfig()
	admin, err := sarama.NewClusterAdmin([]string{testutil.GetTestKafkaHost()}, cfg)
	if err != nil {
		t.Logf("failed to create cluster admin for topic cleanup: %v", err)
		return
	}
	defer admin.Close()

	if err := admin.DeleteTopic(topic); err != nil {
		t.Logf("failed to delete topic %q: %v", topic, err)
	}
}
