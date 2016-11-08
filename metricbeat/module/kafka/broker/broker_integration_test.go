// +build integration
package broker

import (
	"fmt"
	"os"
	"testing"

	"github.com/Shopify/sarama"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
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

	/*f := mbtest.NewEventsFetcher(t, getConfig())
	dataBefore, err := f.Fetch()
	if err != nil {
		t.Fatal("write", err)
	}

	// Create 10 messages
	for i := 0; i < 10; i++ {
		generateKafkaData(t)
	}

	dataAfter, err := f.Fetch()
	if err != nil {
		t.Fatal("write", err)
	}*/

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

	msg := &sarama.ProducerMessage{
		Topic: "testtopic",
		Value: sarama.StringEncoder("Hello World"),
	}

	_, _, err = producer.SendMessage(msg)
	if err != nil {
		t.Errorf("FAILED to send message: %s\n", err)
	}

}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "kafka",
		"metricsets": []string{"topic"},
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
