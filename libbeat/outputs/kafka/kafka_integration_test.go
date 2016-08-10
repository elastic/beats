// +build integration

package kafka

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/outil"
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

	sel := outil.MakeSelector(outil.ConstSelectorExpr(topic))
	client, err := newKafkaClient(hosts, sel, nil)
	if err != nil {
		t.Fatal(err)
	}

	return client
}

func newTestKafkaOutput(t *testing.T, topic string, useType bool) outputs.Outputer {

	if useType {
		topic = "%{[type]}"
	}
	config := map[string]interface{}{
		"hosts":   []string{getTestKafkaHost()},
		"timeout": "1s",
		"topic":   topic,
	}

	cfg, err := common.NewConfigFrom(config)
	if err != nil {
		t.Fatal(err)
	}

	output, err := New("libbeat", cfg, 0)
	if err != nil {
		t.Fatal(err)
	}

	return output
}

func newTestConsumer(t *testing.T) sarama.Consumer {
	hosts := []string{getTestKafkaHost()}
	consumer, err := sarama.NewConsumer(hosts, nil)
	if err != nil {
		t.Fatal(err)
	}
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
	var messages []*sarama.ConsumerMessage
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

	id := strconv.Itoa(rand.New(rand.NewSource(int64(time.Now().Nanosecond()))).Int())
	topic := fmt.Sprintf("test-libbeat-%s", id)

	kafka := newTestKafkaOutput(t, topic, false)
	event := common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"host":       "test-host",
		"type":       "log",
		"message":    id,
	}
	if err := kafka.PublishEvent(nil, testOptions, event); err != nil {
		t.Fatal(err)
	}

	messages := testReadFromKafkaTopic(t, topic, 1, 5*time.Second)
	if assert.Len(t, messages, 1) {
		msg := messages[0]
		logp.Debug("kafka", "%s: %s", msg.Key, msg.Value)
		assert.Contains(t, string(msg.Value), id)
	}
}

func TestUseType(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode. Requires Kafka")
	}
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"kafka"})
	}

	id := strconv.Itoa(rand.New(rand.NewSource(int64(time.Now().Nanosecond()))).Int())
	logType := fmt.Sprintf("log-type-%s", id)

	kafka := newTestKafkaOutput(t, "", true)
	event := common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"host":       "test-host",
		"type":       logType,
		"message":    id,
	}
	if err := kafka.PublishEvent(nil, testOptions, event); err != nil {
		t.Fatal(err)
	}

	messages := testReadFromKafkaTopic(t, logType, 1, 5*time.Second)
	if assert.Len(t, messages, 1) {
		msg := messages[0]
		logp.Debug("kafka", "%s: %s", msg.Key, msg.Value)
		assert.Contains(t, string(msg.Value), id)
	}
}
