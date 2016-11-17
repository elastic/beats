package kafka

import (
	"fmt"
	"os"
	"testing"

	"github.com/Shopify/sarama"
)

const (
	kafkaDefaultHost = "localhost"
	kafkaDefaultPort = "9092"
)

func GenerateKafkaData(t *testing.T) {

	config := sarama.NewConfig()
	client, err := sarama.NewClient([]string{GetTestKafkaHost()}, config)
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

func GetTestKafkaHost() string {
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
