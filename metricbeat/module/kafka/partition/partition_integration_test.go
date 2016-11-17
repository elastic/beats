// +build integration

package partition

import (
	"fmt"
	"os"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/stretchr/testify/assert"
)

const (
	kafkaDefaultHost = "localhost"
	kafkaDefaultPort = "9092"
)

func TestData(t *testing.T) {

	generateKafkaData(t)

	f := mbtest.NewEventsFetcher(t, getConfig())
	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func TestTopic(t *testing.T) {

	// Create initial topic
	generateKafkaData(t)

	f := mbtest.NewEventsFetcher(t, getConfig())
	dataBefore, err := f.Fetch()
	if err != nil {
		t.Fatal("write", err)
	}

	var n int64 = 10
	var i int64 = 0
	// Create n messages
	for ; i < n; i++ {
		generateKafkaData(t)
	}

	dataAfter, err := f.Fetch()
	if err != nil {
		t.Fatal("write", err)
	}

	// Checks that no new topics / partitions were added
	assert.True(t, len(dataBefore) == len(dataAfter))

	// Compares offset before and after
	offsetBefore := dataBefore[0]["offset"].(common.MapStr)["newest"].(int64)
	offsetAfter := dataAfter[0]["offset"].(common.MapStr)["newest"].(int64)

	if offsetBefore+n != offsetAfter {
		t.Errorf("Offset before: %v", offsetBefore)
		t.Errorf("Offset after: %v", offsetAfter)
	}
	assert.True(t, offsetBefore+n == offsetAfter)

}

func generateKafkaData(t *testing.T) {

	config := sarama.NewConfig()
	client, err := sarama.NewClient([]string{getTestKafkaHost()}, config)
	if err != nil {
		t.Errorf("%s", err)
	}

	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		t.Error(err)
	}
	defer producer.Close()

	topic := "testtopic"

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

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "kafka",
		"metricsets": []string{"partition"},
		"hosts":      []string{getTestKafkaHost()},
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
