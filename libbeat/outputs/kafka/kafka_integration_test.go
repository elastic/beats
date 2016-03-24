// +build integration

package kafka

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/stretchr/testify/assert"
)

const (
	kafkaDefaultHost = "localhost"
	kafkaDefaultPort = "9092"
)

var testOptions = outputs.Options{}

func strDefault(a, defaults string) string {
	if len(a) == 0 {
		return defaults
	}
	return a
}

func getenv(name, defaultValue string) string {
	return strDefault(os.Getenv(name), defaultValue)
}

func getTestKafkaHost() string {
	return fmt.Sprintf("%v:%v",
		getenv("KAFKA_HOST", kafkaDefaultHost),
		getenv("KAFKA_PORT", kafkaDefaultPort),
	)
}

func newTestKafkaClient(t *testing.T, topic string) *client {

	hosts := []string{getTestKafkaHost()}
	t.Logf("host: %v", hosts)

	client, err := newKafkaClient(hosts, topic, false, nil)
	assert.NoError(t, err)

	return client
}

func newTestKafkaOutput(t *testing.T, topic string, useType bool) outputs.Outputer {

	config := map[string]interface{}{
		"hosts":    []string{getTestKafkaHost()},
		"timeout":  "1s",
		"topic":    topic,
		"use_type": useType,
	}

	cfg, err := common.NewConfigFrom(config)
	assert.NoError(t, err)
	output, err := New(cfg, 0)
	assert.NoError(t, err)

	return output
}

func newTestConsumer(t *testing.T) sarama.Consumer {
	hosts := []string{getTestKafkaHost()}
	consumer, err := sarama.NewConsumer(hosts, nil)
	assert.NoError(t, err)
	return consumer
}

func testReadFromKafkaTopic(
	t *testing.T, topic string, nMessages int,
	timeout time.Duration) []*sarama.ConsumerMessage {

	consumer := newTestConsumer(t)
	defer func() {
		consumer.Close()
	}()

	partitionConsumer, err := consumer.ConsumePartition(topic, 0, sarama.OffsetOldest)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		partitionConsumer.Close()
	}()

	timer := time.After(timeout)
	messages := []*sarama.ConsumerMessage{}
	for i := 0; i < nMessages; i++ {
		select {
		case msg := <-partitionConsumer.Messages():
			messages = append(messages, msg)
		case <-timer:
			break
		}

	}

	return messages
}

func TestOneMessageToKafka(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode. Requires Kafka")
	}
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"kafka"})
	}

	kafka := newTestKafkaOutput(t, "test-libbeat", false)

	event := common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"host":       "test-host",
		"type":       "log",
		"message":    "hello world",
	}
	err := kafka.PublishEvent(nil, testOptions, event)
	assert.NoError(t, err)

	messages := testReadFromKafkaTopic(t, "test-libbeat", 1, 5*time.Second)

	msg := messages[0]
	logp.Debug("kafka", "%s: %s", msg.Key, msg.Value)
}

func TestUseType(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode. Requires Kafka")
	}
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"kafka"})
	}

	kafka := newTestKafkaOutput(t, "", true)

	event := common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"host":       "test-host",
		"type":       "log-type",
		"message":    "hello world",
	}
	err := kafka.PublishEvent(nil, testOptions, event)
	assert.NoError(t, err)

	messages := testReadFromKafkaTopic(t, "log-type", 1, 5*time.Second)

	msg := messages[0]
	logp.Debug("kafka", "%s: %s", msg.Key, msg.Value)
}
