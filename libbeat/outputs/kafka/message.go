package kafka

import (
	"time"

	"github.com/Shopify/sarama"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
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

	event common.MapStr
}

var kafkaMessageKey interface{} = int(0)

func messageFromData(d *outputs.Data) *message {
	if m, found := d.Values.Get(kafkaMessageKey); found {
		return m.(*message)
	}

	m := &message{partition: -1}
	d.AddValue(kafkaMessageKey, m)
	return m
}

func (m *message) initProducerMessage() {
	m.msg = sarama.ProducerMessage{
		Metadata:  m,
		Topic:     m.topic,
		Key:       sarama.ByteEncoder(m.key),
		Value:     sarama.ByteEncoder(m.value),
		Timestamp: m.ts,
	}
}
