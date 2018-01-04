// +build integration

package partition

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

const (
	kafkaDefaultHost = "localhost"
	kafkaDefaultPort = "9092"
)

func TestData(t *testing.T) {
	compose.EnsureUp(t, "kafka")

	generateKafkaData(t, "metricbeat-generate-data")

	f := mbtest.NewEventsFetcher(t, getConfig(""))
	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func TestTopic(t *testing.T) {

	compose.EnsureUp(t, "kafka")

	logp.TestingSetup(logp.WithSelectors("kafka"))

	id := strconv.Itoa(rand.New(rand.NewSource(int64(time.Now().Nanosecond()))).Int())
	testTopic := fmt.Sprintf("test-metricbeat-%s", id)

	// Create initial topic
	generateKafkaData(t, testTopic)

	f := mbtest.NewEventsFetcher(t, getConfig(testTopic))
	dataBefore, err := f.Fetch()
	if err != nil {
		t.Fatal("write", err)
	}
	if len(dataBefore) == 0 {
		t.Errorf("No offsets fetched from topic (before): %v", testTopic)
	}
	t.Logf("before: %v", dataBefore)

	var n int64 = 10
	var i int64 = 0
	// Create n messages
	for ; i < n; i++ {
		generateKafkaData(t, testTopic)
	}

	dataAfter, err := f.Fetch()
	if err != nil {
		t.Fatal("write", err)
	}
	if len(dataAfter) == 0 {
		t.Errorf("No offsets fetched from topic (after): %v", testTopic)
	}
	t.Logf("after: %v", dataAfter)

	// Checks that no new topics / partitions were added
	assert.True(t, len(dataBefore) == len(dataAfter))

	var offsetBefore int64 = 0
	var offsetAfter int64 = 0

	// Its possible that other topics exists -> select the right data
	for _, data := range dataBefore {
		if data["topic"].(common.MapStr)["name"] == testTopic {
			offsetBefore = data["offset"].(common.MapStr)["newest"].(int64)
		}
	}

	for _, data := range dataAfter {
		if data["topic"].(common.MapStr)["name"] == testTopic {
			offsetAfter = data["offset"].(common.MapStr)["newest"].(int64)
		}
	}

	// Compares offset before and after
	if offsetBefore+n != offsetAfter {
		t.Errorf("Offset before: %v", offsetBefore)
		t.Errorf("Offset after: %v", offsetAfter)
	}
	assert.True(t, offsetBefore+n == offsetAfter)
}

func generateKafkaData(t *testing.T, topic string) {
	t.Logf("Send Kafka Event to topic: %v", topic)

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	// Retry for 10 seconds
	config.Producer.Retry.Max = 20
	config.Producer.Retry.Backoff = 500 * time.Millisecond
	config.Metadata.Retry.Max = 20
	config.Metadata.Retry.Backoff = 500 * time.Millisecond
	client, err := sarama.NewClient([]string{getTestKafkaHost()}, config)
	if err != nil {
		t.Errorf("%s", err)
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
		t.Errorf("FAILED to send message: %s\n", err)
	}

	client.RefreshMetadata(topic)
}

func getConfig(topic string) map[string]interface{} {
	var topics []string
	if topic != "" {
		topics = []string{topic}
	}

	return map[string]interface{}{
		"module":     "kafka",
		"metricsets": []string{"partition"},
		"hosts":      []string{getTestKafkaHost()},
		"topics":     topics,
	}
}

func getTestKafkaHost() string {
	return fmt.Sprintf("%v:%v",
		getenv("KAFKA_HOST", kafkaDefaultHost),
		getenv("KAFKA_PORT", kafkaDefaultPort),
	)
}

func getenv(name, defaultValue string) string {
	return strDefault(os.Getenv(name), defaultValue)
}

func strDefault(a, defaults string) string {
	if len(a) == 0 {
		return defaults
	}
	return a
}
