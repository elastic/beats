package kafka

import (
	"time"

	"github.com/Shopify/sarama"

	"github.com/elastic/beats/libbeat/publisher"
)

type message struct {
	msg sarama.ProducerMessage

	topic string
	key   []byte
	value []byte
	ref   *msgRef
	ts    time.Time

	hash      uint32
	partition int32

	data publisher.Event
}

var kafkaMessageKey interface{} = int(0)

func (m *message) initProducerMessage() {
	m.msg = sarama.ProducerMessage{
		Metadata:  m,
		Topic:     m.topic,
		Key:       sarama.ByteEncoder(m.key),
		Value:     sarama.ByteEncoder(m.value),
		Timestamp: m.ts,
	}
}
