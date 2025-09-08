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
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/json"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/sarama"
)

func TestClientShutdownPanic(t *testing.T) {
	logger, buff := logp.NewInMemoryLocal("", logp.ConsoleEncoderConfig())

	cfg, err := config.NewConfigFrom(map[string]interface{}{
		"hosts":   []string{"localhost:9094"},
		"topic":   "testTopic",
		"timeout": "1s",
	})
	require.NoError(t, err, "could not create config")

	outGroup, err := makeKafka(
		nil,
		beat.Info{
			Beat:        "libbeat",
			IndexPrefix: "testbeat",
			Logger:      logger},
		outputs.NewStats(monitoring.NewRegistry(), logger), cfg)
	require.NoError(t, err, "could not create kafka output")

	b := outest.NewBatch(
		beat.Event{
			Timestamp:  time.Time{},
			Meta:       nil,
			Fields:     map[string]any{"msg": "message 1"},
			Private:    nil,
			TimeSeries: false,
		},
		beat.Event{
			Timestamp:  time.Time{},
			Meta:       nil,
			Fields:     map[string]any{"msg": "message 2"},
			Private:    nil,
			TimeSeries: false,
		})

	ch := make(chan *sarama.ProducerMessage)
	wc := sync.WaitGroup{}

	c, ok := outGroup.Clients[0].(*client)
	require.Truef(t, ok, "Expected output to be of type %T", &client{})

	c.producer = producerMock{input: ch}

	// 1st: Publish and block on channel send
	wc.Add(1)
	go func() {
		defer wc.Done()
		err := c.Publish(context.Background(), b)
		require.NoError(t, err, "publish failed")
	}()

	// 2nd: Get 1st message to make sure the Publishing goroutine run and did
	// all it needs before sending the messages to the channel
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("publish never sent 1st message")
	}

	// 3rd: Close the client while the Publish is blocked waiting to send the
	// 2nd event
	err = c.Close()
	require.NoError(t, err, "close failed")

	// 4th: wait the publishing goroutine to attempt to publish the 2nd event
	wc.Wait()

	// 5th. assert the event dropped log is there:
	assert.Contains(t, buff.String(), "output closing, dropping event",
		"event dropped log not found")
}

type producerMock struct {
	input chan *sarama.ProducerMessage
}

func (p producerMock) AsyncClose() {
	close(p.input)
}

func (p producerMock) Input() chan<- *sarama.ProducerMessage {
	return p.input
}

func (p producerMock) Close() error {
	// TODO implement me
	panic("implement me")
}

func (p producerMock) Successes() <-chan *sarama.ProducerMessage {
	// TODO implement me
	panic("implement me")
}

func (p producerMock) Errors() <-chan *sarama.ProducerError {
	// TODO implement me
	panic("implement me")
}

func (p producerMock) IsTransactional() bool {
	// TODO implement me
	panic("implement me")
}

func (p producerMock) TxnStatus() sarama.ProducerTxnStatusFlag {
	// TODO implement me
	panic("implement me")
}

func (p producerMock) BeginTxn() error {
	// TODO implement me
	panic("implement me")
}

func (p producerMock) CommitTxn() error {
	// TODO implement me
	panic("implement me")
}

func (p producerMock) AbortTxn() error {
	// TODO implement me
	panic("implement me")
}

func (p producerMock) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupId string) error {
	// TODO implement me
	panic("implement me")
}

func (p producerMock) AddMessageToTxn(msg *sarama.ConsumerMessage, groupId string, metadata *string) error {
	// TODO implement me
	panic("implement me")
}
