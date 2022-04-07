// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package kafka

import (
	"time"

	"github.com/Shopify/sarama"

	"github.com/elastic/beats/v7/libbeat/publisher"
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

	if m.ref != nil {
		m.msg.Headers = m.ref.client.recordHeaders
	}
}
