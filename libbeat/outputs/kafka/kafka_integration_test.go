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

//go:build integration
// +build integration

package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/format"
	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/json"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
)

const (
	kafkaDefaultHost     = "kafka"
	kafkaDefaultPort     = "9092"
	kafkaDefaultSASLPort = "9093"
)

type eventInfo struct {
	events []beat.Event
}

func TestKafkaPublish(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("kafka"))

	id := strconv.Itoa(rand.New(rand.NewSource(int64(time.Now().Nanosecond()))).Int())
	testTopic := fmt.Sprintf("test-libbeat-%s", id)
	logType := fmt.Sprintf("log-type-%s", id)

	tests := []struct {
		title  string
		config map[string]interface{}
		topic  string
		events []eventInfo
	}{
		{
			"publish single event to test topic",
			nil,
			testTopic,
			single(common.MapStr{
				"host":    "test-host",
				"message": id,
			}),
		},
		{
			"publish single event with topic from type",
			map[string]interface{}{
				"topic": "%{[type]}",
			},
			logType,
			single(common.MapStr{
				"host":    "test-host",
				"type":    logType,
				"message": id,
			}),
		},
		{
			"publish single event with formating to test topic",
			map[string]interface{}{
				"codec.format.string": "%{[message]}",
			},
			testTopic,
			single(common.MapStr{
				"host":    "test-host",
				"message": id,
			}),
		},
		{
			"batch publish to test topic",
			nil,
			testTopic,
			randMulti(5, 100, common.MapStr{
				"host": "test-host",
			}),
		},
		{
			"batch publish to test topic from type",
			map[string]interface{}{
				"topic": "%{[type]}",
			},
			logType,
			randMulti(5, 100, common.MapStr{
				"host": "test-host",
				"type": logType,
			}),
		},
		{
			"batch publish with random partitioner",
			map[string]interface{}{
				"partition.random": map[string]interface{}{
					"group_events": 1,
				},
			},
			testTopic,
			randMulti(1, 10, common.MapStr{
				"host": "test-host",
				"type": "log",
			}),
		},
		{
			"batch publish with round robin partitioner",
			map[string]interface{}{
				"partition.round_robin": map[string]interface{}{
					"group_events": 1,
				},
			},
			testTopic,
			randMulti(1, 10, common.MapStr{
				"host": "test-host",
				"type": "log",
			}),
		},
		{
			"batch publish with hash partitioner without key (fallback to random)",
			map[string]interface{}{
				"partition.hash": map[string]interface{}{},
			},
			testTopic,
			randMulti(1, 10, common.MapStr{
				"host": "test-host",
				"type": "log",
			}),
		},
		{
			// warning: this test uses random keys. In case keys are reused, test might fail.
			"batch publish with hash partitioner with key",
			map[string]interface{}{
				"key":            "%{[message]}",
				"partition.hash": map[string]interface{}{},
			},
			testTopic,
			randMulti(1, 10, common.MapStr{
				"host": "test-host",
				"type": "log",
			}),
		},
		{
			// warning: this test uses random keys. In case keys are reused, test might fail.
			"batch publish with fields hash partitioner",
			map[string]interface{}{
				"partition.hash.hash": []string{
					"@timestamp",
					"type",
					"message",
				},
			},
			testTopic,
			randMulti(1, 10, common.MapStr{
				"host": "test-host",
				"type": "log",
			}),
		},
		{
			"publish single event to test topic",
			map[string]interface{}{},
			testTopic,
			single(common.MapStr{
				"host":    "test-host",
				"message": id,
			}),
		},
		{
			// Initially I tried rerunning all tests over SASL/SCRAM, but
			// that added a full 30sec to the test. Instead most tests run
			// in plaintext, and individual tests can switch to SCRAM
			// by inserting the config in this example:
			"publish single event to test topic over SASL/SCRAM",
			map[string]interface{}{
				"hosts":          []string{getTestSASLKafkaHost()},
				"protocol":       "https",
				"sasl.mechanism": "SCRAM-SHA-512",
				"ssl.certificate_authorities": []string{
					"../../../testing/environments/docker/kafka/certs/ca-cert",
				},
				"username": "beats",
				"password": "KafkaTest",
			},
			testTopic,
			single(common.MapStr{
				"host":    "test-host",
				"message": id,
			}),
		},
		{
			"publish message with kafka headers to test topic",
			map[string]interface{}{
				"headers": map[string]string{
					"app":  "test-app",
					"host": "test-host",
				},
			},
			testTopic,
			randMulti(5, 100, common.MapStr{
				"host": "test-host",
			}),
		},
	}

	defaultConfig := map[string]interface{}{
		"hosts":   []string{getTestKafkaHost()},
		"topic":   testTopic,
		"timeout": "1s",
	}

	for i, test := range tests {
		test := test
		name := fmt.Sprintf("run test(%v): %v", i, test.title)

		cfg := makeConfig(t, defaultConfig)
		if test.config != nil {
			cfg.Merge(makeConfig(t, test.config))
		}

		t.Run(name, func(t *testing.T) {
			grp, err := makeKafka(nil, beat.Info{Beat: "libbeat", IndexPrefix: "testbeat"}, outputs.NewNilObserver(), cfg)
			if err != nil {
				t.Fatal(err)
			}

			output := grp.Clients[0].(*client)
			if err := output.Connect(); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, output.index, "testbeat")
			defer output.Close()

			// publish test events
			var wg sync.WaitGroup
			for i := range test.events {
				batch := outest.NewBatch(test.events[i].events...)
				batch.OnSignal = func(_ outest.BatchSignal) {
					wg.Done()
				}

				wg.Add(1)
				output.Publish(context.Background(), batch)
			}

			// wait for all published batches to be ACKed
			wg.Wait()

			expected := flatten(test.events)

			// check we can find all event in topic
			timeout := 20 * time.Second
			stored := testReadFromKafkaTopic(t, test.topic, len(expected), timeout)

			// validate messages
			if len(expected) != len(stored) {
				assert.Equal(t, len(stored), len(expected))
				return
			}

			validate := validateJSON
			if fmt, exists := test.config["codec.format.string"]; exists {
				validate = makeValidateFmtStr(fmt.(string))
			}

			seenMsgs := map[string]struct{}{}
			for _, s := range stored {
				if headers, exists := test.config["headers"]; exists {
					assert.Equal(t, len(headers.(map[string]string)), len(s.Headers))
				}

				msg := validate(t, s.Value, expected)
				seenMsgs[msg] = struct{}{}
			}
			assert.Equal(t, len(expected), len(seenMsgs))
		})
	}
}

func validateJSON(t *testing.T, value []byte, events []beat.Event) string {
	var decoded map[string]interface{}
	err := json.Unmarshal(value, &decoded)
	if err != nil {
		t.Errorf("can not json decode event value: %v", value)
		return ""
	}

	msg := decoded["message"].(string)
	event := findEvent(events, msg)
	if event == nil {
		t.Errorf("could not find expected event with message: %v", msg)
		return ""
	}

	assert.Equal(t, decoded["type"], event.Fields["type"])

	return msg
}

func makeValidateFmtStr(fmt string) func(*testing.T, []byte, []beat.Event) string {
	fmtString := fmtstr.MustCompileEvent(fmt)
	return func(t *testing.T, value []byte, events []beat.Event) string {
		msg := string(value)
		event := findEvent(events, msg)
		if event == nil {
			t.Errorf("could not find expected event with message: %v", msg)
			return ""
		}

		_, err := fmtString.Run(event)
		if err != nil {
			t.Fatal(err)
		}

		return msg
	}
}

func findEvent(events []beat.Event, msg string) *beat.Event {
	for _, e := range events {
		if e.Fields["message"] == msg {
			return &e
		}
	}

	return nil
}

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

func getTestSASLKafkaHost() string {
	return fmt.Sprintf("%v:%v",
		getenv("KAFKA_HOST", kafkaDefaultHost),
		getenv("KAFKA_SASL_PORT", kafkaDefaultSASLPort),
	)
}

func makeConfig(t *testing.T, in map[string]interface{}) *common.Config {
	cfg, err := common.NewConfigFrom(in)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func newTestConsumer(t *testing.T) sarama.Consumer {
	hosts := []string{getTestKafkaHost()}
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V1_0_0_0
	consumer, err := sarama.NewConsumer(hosts, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return consumer
}

// topicOffsetMap is threadsafe map from topic => partition => offset
type topicOffsetMap struct {
	m  map[string]map[int32]int64
	mu sync.RWMutex
}

func (m *topicOffsetMap) GetOffset(topic string, partition int32) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.m == nil {
		return sarama.OffsetOldest
	}

	topicMap, ok := m.m[topic]
	if !ok {
		return sarama.OffsetOldest
	}

	offset, ok := topicMap[partition]
	if !ok {
		return sarama.OffsetOldest
	}

	return offset
}

func (m *topicOffsetMap) SetOffset(topic string, partition int32, offset int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.m == nil {
		m.m = map[string]map[int32]int64{}
	}

	if _, ok := m.m[topic]; !ok {
		m.m[topic] = map[int32]int64{}
	}

	m.m[topic][partition] = offset
}

var testTopicOffsets = topicOffsetMap{}

func testReadFromKafkaTopic(
	t *testing.T, topic string, nMessages int,
	timeout time.Duration,
) []*sarama.ConsumerMessage {
	consumer := newTestConsumer(t)
	defer func() {
		consumer.Close()
	}()

	partitions, err := consumer.Partitions(topic)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	msgs := make(chan *sarama.ConsumerMessage)
	for _, partition := range partitions {
		offset := testTopicOffsets.GetOffset(topic, partition)
		partitionConsumer, err := consumer.ConsumePartition(topic, partition, offset)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			partitionConsumer.Close()
		}()

		go func(p int32, pc sarama.PartitionConsumer) {
			for {
				select {
				case msg, ok := <-pc.Messages():
					if !ok {
						break
					}
					testTopicOffsets.SetOffset(topic, p, msg.Offset+1)
					msgs <- msg
				case <-done:
					break
				}
			}
		}(partition, partitionConsumer)
	}

	var messages []*sarama.ConsumerMessage
	timer := time.After(timeout)

	for len(messages) < nMessages {
		select {
		case msg := <-msgs:
			messages = append(messages, msg)
		case <-timer:
			break
		}
	}

	close(done)
	return messages
}

func flatten(infos []eventInfo) []beat.Event {
	var out []beat.Event
	for _, info := range infos {
		out = append(out, info.events...)
	}
	return out
}

func single(fields common.MapStr) []eventInfo {
	return []eventInfo{
		{
			events: []beat.Event{
				{Timestamp: time.Now(), Fields: fields},
			},
		},
	}
}

func randMulti(batches, n int, event common.MapStr) []eventInfo {
	var out []eventInfo
	for i := 0; i < batches; i++ {
		var data []beat.Event
		for j := 0; j < n; j++ {
			tmp := common.MapStr{}
			for k, v := range event {
				tmp[k] = v
			}
			tmp["message"] = randString(100)
			data = append(data, beat.Event{Timestamp: time.Now(), Fields: tmp})
		}

		out = append(out, eventInfo{data})
	}
	return out
}
