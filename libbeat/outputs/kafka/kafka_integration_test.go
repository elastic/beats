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

package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math/rand/v2"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/sarama"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/outputs"
	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/format"
	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/json"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	kafkaDefaultHost     = "localhost"
	kafkaDefaultPort     = "9094"
	kafkaDefaultSASLPort = "9093"
)

type eventInfo struct {
	events []beat.Event
}

func TestKafkaPublish(t *testing.T) {

	id := strconv.Itoa(rand.Int())
	testTopic := fmt.Sprintf("test-libbeat-%s", id)
	logType := fmt.Sprintf("log-type-%s", id)

	// Topic auto-creation is asynchronous; wait for leaders before publishing to avoid
	// "no leader for this partition" failures during leadership election.
	ensureKafkaTopicReadyForWrites(t, testTopic)
	ensureKafkaTopicReadyForWrites(t, logType)

	tests := []struct {
		title  string
		config map[string]any
		topic  string
		events []eventInfo
	}{
		{
			"publish single event to test topic with nil config",
			nil,
			testTopic,
			single(mapstr.M{
				"host":    "test-host",
				"message": id,
			}),
		},
		{
			"publish single event with topic from type",
			map[string]any{
				"topic": "%{[type]}",
			},
			logType,
			single(mapstr.M{
				"host":    "test-host",
				"type":    logType,
				"message": id,
			}),
		},
		{
			"publish single event with formating to test topic",
			map[string]any{
				"codec.format.string": "%{[message]}",
			},
			testTopic,
			single(mapstr.M{
				"host":    "test-host",
				"message": id,
			}),
		},
		{
			"batch publish to test topic",
			nil,
			testTopic,
			randMulti(5, 100, mapstr.M{
				"host": "test-host",
			}),
		},
		{
			"batch publish to test topic from type",
			map[string]any{
				"topic": "%{[type]}",
			},
			logType,
			randMulti(5, 100, mapstr.M{
				"host": "test-host",
				"type": logType,
			}),
		},
		{
			"batch publish with random partitioner",
			map[string]any{
				"partition.random": map[string]any{
					"group_events": 1,
				},
			},
			testTopic,
			randMulti(1, 10, mapstr.M{
				"host": "test-host",
				"type": "log",
			}),
		},
		{
			"batch publish with round robin partitioner",
			map[string]any{
				"partition.round_robin": map[string]any{
					"group_events": 1,
				},
			},
			testTopic,
			randMulti(1, 10, mapstr.M{
				"host": "test-host",
				"type": "log",
			}),
		},
		{
			"batch publish with hash partitioner without key (fallback to random)",
			map[string]any{
				"partition.hash": map[string]any{},
			},
			testTopic,
			randMulti(1, 10, mapstr.M{
				"host": "test-host",
				"type": "log",
			}),
		},
		{
			// warning: this test uses random keys. In case keys are reused, test might fail.
			"batch publish with hash partitioner with key",
			map[string]any{
				"key":            "%{[message]}",
				"partition.hash": map[string]any{},
			},
			testTopic,
			randMulti(1, 10, mapstr.M{
				"host": "test-host",
				"type": "log",
			}),
		},
		{
			// warning: this test uses random keys. In case keys are reused, test might fail.
			"batch publish with fields hash partitioner",
			map[string]any{
				"partition.hash.hash": []string{
					"@timestamp",
					"type",
					"message",
				},
			},
			testTopic,
			randMulti(1, 10, mapstr.M{
				"host": "test-host",
				"type": "log",
			}),
		},
		{
			"publish single event to test topic with empty config",
			map[string]any{},
			testTopic,
			single(mapstr.M{
				"host":    "test-host",
				"message": id,
			}),
		},
		{
			// Initially I tried rerunning all tests over SASL/SCRAM, but
			// that added a full 30sec to the test. Instead most tests run
			// in plaintext, and individual tests can switch to SCRAM
			// by inserting the config in this example:
			"SASL/SCRAM publish single event to test topic",
			map[string]any{
				"hosts":          []string{getTestSASLKafkaHost()},
				"protocol":       "https",
				"sasl.mechanism": "SCRAM-SHA-512",
				// Disable hostname verification since we are likely writing to localhost.
				"ssl.verification_mode": "certificate",
				"ssl.certificate_authorities": []string{
					"../../../testing/environments/docker/kafka/certs/ca-cert",
				},
				"username": "beats",
				"password": "KafkaTest",
			},
			testTopic,
			single(mapstr.M{
				"host":    "test-host",
				"message": id,
			}),
		},
		{
			"publish message with kafka headers to test topic",
			map[string]any{
				"headers": []map[string]string{
					{
						"key":   "app",
						"value": "test-app",
					},
					{
						"key":   "app",
						"value": "test-app2",
					},
					{
						"key":   "host",
						"value": "test-host",
					},
				},
			},
			testTopic,
			randMulti(5, 100, mapstr.M{
				"host": "test-host",
			}),
		},
		{
			"publish message with zstd compression to test topic",
			map[string]any{
				"compression": "zstd",
				"version":     "2.2",
			},
			testTopic,
			single(mapstr.M{
				"host":    "test-host",
				"message": id,
			}),
		},
	}

	defaultConfig := map[string]any{
		"hosts":   []string{getTestKafkaHost()},
		"topic":   testTopic,
		"timeout": "1s",
	}

	for i, test := range tests {
		name := fmt.Sprintf("run test(%v): %v", i, test.title)

		cfg := makeConfig(t, defaultConfig)
		if test.config != nil {
			err := cfg.Merge(makeConfig(t, test.config))
			if err != nil {
				t.Fatal(err)
			}
		}

		t.Run(name, func(t *testing.T) {
			logger := logptest.NewTestingLogger(t, "")
			grp, err := makeKafka(nil, beat.Info{Beat: "libbeat", IndexPrefix: "testbeat", Logger: logger}, outputs.NewNilObserver(), cfg)
			if err != nil {
				t.Fatal(err)
			}

			output, ok := grp.Clients[0].(*client)
			assert.True(t, ok, "grp.Clients[0] didn't contain a ptr to client")
			if err := output.Connect(context.Background()); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "testbeat", output.index)
			defer output.Close()

			// Publish asynchronously; OnSignal runs for any terminal outcome (ACK, retry, drop, ...).
			var wg sync.WaitGroup
			batches := make([]*outest.Batch, 0, len(test.events))
			for i := range test.events {
				batch := outest.NewBatch(test.events[i].events...)
				batches = append(batches, batch)
				batch.OnSignal = func(_ outest.BatchSignal) {
					wg.Done()
				}

				wg.Add(1)
				err := output.Publish(context.Background(), batch)
				if err != nil {
					t.Fatal(err)
				}
			}

			// Wait until the output has finished handling every batch (not necessarily ACKed).
			wg.Wait()
			// Failed publishes signal BatchRetryEvents; only proceed to read the topic after ACK.
			requiretBatchesACKed(t, batches)

			expected := flatten(test.events)

			// check we can find all event in topic
			timeout := 20 * time.Second
			stored := testReadFromKafkaTopic(t, test.topic, len(expected), timeout)

			// validate messages
			if len(expected) != len(stored) {
				assert.Len(t, stored, len(expected))
				return
			}

			validate := validateJSON
			if fmt, exists := test.config["codec.format.string"]; exists {
				validate = makeValidateFmtStr(fmt.(string)) //nolint:errcheck //This is a test file
			}

			cfgHeaders, headersSet := test.config["headers"]

			seenMsgs := map[string]struct{}{}
			for _, s := range stored {
				if headersSet {
					expectedHeaders, ok := cfgHeaders.([]map[string]string)
					assert.True(t, ok)
					assert.Len(t, s.Headers, len(expectedHeaders))
					for i, h := range s.Headers {
						expectedHeader := expectedHeaders[i]
						key := string(h.Key)
						value := string(h.Value)
						assert.Equal(t, expectedHeader["key"], key)
						assert.Equal(t, expectedHeader["value"], value)
					}
				}

				msg := validate(t, s.Value, expected)
				seenMsgs[msg] = struct{}{}
			}
			assert.Len(t, seenMsgs, len(expected))
		})
	}
}

func TestKafkaErrors(t *testing.T) {
	id := strconv.Itoa(rand.Int())
	testTopic := fmt.Sprintf("test-libbeat-%s", id)
	ensureKafkaTopicReadyForWrites(t, testTopic)

	tests := []struct {
		title        string
		config       map[string]any
		topic        string
		events       []eventInfo
		errorMessage string
	}{
		{
			"message of size large than `max_message_bytes` must be dropped",
			map[string]any{
				"max_message_bytes": "10",
			},
			testTopic,
			single(mapstr.M{
				"host":    "test-host-random-message-which-is-long-enough",
				"message": id,
			}),
			"dropping message as it exceeds max_mesage_bytes",
		},
	}

	defaultConfig := map[string]any{
		"hosts":   []string{getTestKafkaHost()},
		"topic":   testTopic,
		"timeout": "1s",
	}

	for _, test := range tests {

		cfg := makeConfig(t, defaultConfig)
		if test.config != nil {
			err := cfg.Merge(makeConfig(t, test.config))
			if err != nil {
				t.Fatal(err)
			}
		}

		observed, zapLogs := observer.New(zapcore.DebugLevel)
		logger, err := logp.ConfigureWithCoreLocal(logp.Config{}, observed)
		require.NoError(t, err)

		grp, err := makeKafka(nil, beat.Info{Beat: "libbeat", IndexPrefix: "testbeat", Logger: logger}, outputs.NewNilObserver(), cfg)
		if err != nil {
			t.Fatal(err)
		}

		output, ok := grp.Clients[0].(*client)
		assert.True(t, ok, "grp.Clients[0] didn't contain a ptr to client")
		if err := output.Connect(context.Background()); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "testbeat", output.index)
		defer output.Close()

		// publish test events
		var wg sync.WaitGroup
		for i := range test.events {
			batch := outest.NewBatch(test.events[i].events...)
			batch.OnSignal = func(_ outest.BatchSignal) {
				wg.Done()
			}

			wg.Add(1)
			err := output.Publish(context.Background(), batch)
			if err != nil {
				t.Fatal(err)
			}
		}

		// Wait until handling completes; this test expects a drop/retry, not necessarily ACK.
		wg.Wait()

		t.Cleanup(func() {
			if t.Failed() {
				t.Logf("Debug Logs:\n")
				for _, log := range zapLogs.TakeAll() {
					data, err := json.Marshal(log)
					if err != nil {
						t.Errorf("failed encoding log as JSON: %s", err)
					}
					t.Logf("%s", string(data))
				}
				return
			}
		})
		assert.GreaterOrEqual(t, zapLogs.FilterMessageSnippet(test.errorMessage).Len(), 1)
	}

}

func validateJSON(t *testing.T, value []byte, events []beat.Event) string {
	var decoded map[string]any
	err := json.Unmarshal(value, &decoded)
	if err != nil {
		t.Errorf("can not json decode event value: %v", value)
		return ""
	}

	msg, ok := decoded["message"].(string)
	assert.True(t, ok, "type of decoded message was not string")
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

func ensureKafkaTopicReadyForWrites(t *testing.T, topic string) {
	t.Helper()

	saramaCfg := sarama.NewConfig()
	saramaCfg.Version = sarama.V2_1_0_0
	hosts := []string{getTestKafkaHost()}

	admin, err := sarama.NewClusterAdmin(hosts, saramaCfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, admin.Close())
	})

	topicDetail := &sarama.TopicDetail{
		NumPartitions:     3,
		ReplicationFactor: 1,
	}
	require.EventuallyWithTf(t, func(ct *assert.CollectT) {
		err = admin.CreateTopic(topic, topicDetail, false)
		if err != nil && !errors.Is(err, sarama.ErrTopicAlreadyExists) {
			require.NoError(ct, err)
		}
	}, 30*time.Second, 200*time.Millisecond, "failed to create topic %s", topic)

	client, err := sarama.NewClient(hosts, saramaCfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})

	require.EventuallyWithTf(t, func(ct *assert.CollectT) {
		require.NoError(ct, client.RefreshMetadata(topic))

		partitions, err := client.Partitions(topic)
		require.NoError(ct, err)
		require.NotEmpty(ct, partitions)

		for _, partition := range partitions {
			leader, err := client.Leader(topic, partition)
			require.NoError(ct, err)
			require.NotNil(ct, leader)
		}
	}, 30*time.Second, 200*time.Millisecond, "topic %s is not ready for writes", topic)
}

func makeConfig(t *testing.T, in map[string]any) *config.C {
	cfg, err := config.NewConfigFrom(in)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func newTestConsumer(t *testing.T) sarama.Consumer {
	hosts := []string{getTestKafkaHost()}
	consumer, err := sarama.NewConsumer(hosts, nil)
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

// testReadFromKafkaTopic consumes up to nMessages from topic across all partitions.
// Returns after timeout even when fewer than nMessages were read (caller must assert count).
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

readLoop:
	for len(messages) < nMessages {
		select {
		case msg := <-msgs:
			messages = append(messages, msg)
		case <-timer:
			break readLoop
		}
	}

	close(done)
	return messages
}

// requiretBatchesACKed fails if the kafka client finished the batch with retry/drop instead of ACK.
func requiretBatchesACKed(t *testing.T, batches []*outest.Batch) {
	t.Helper()
	for _, batch := range batches {
		require.NotEmpty(t, batch.Signals, "expected publish batch to receive a signal")
		last := batch.Signals[len(batch.Signals)-1]
		require.Equal(t, outest.BatchACK, last.Tag, "expected batch to be ACKed, got %v", last.Tag)
	}
}

func flatten(infos []eventInfo) []beat.Event {
	var out []beat.Event
	for _, info := range infos {
		out = append(out, info.events...)
	}
	return out
}

func single(fields mapstr.M) []eventInfo {
	return []eventInfo{
		{
			events: []beat.Event{
				{Timestamp: time.Now(), Fields: fields},
			},
		},
	}
}

func randMulti(batches, n int, event mapstr.M) []eventInfo {
	var out []eventInfo
	for range batches {
		var data []beat.Event
		for range n {
			tmp := mapstr.M{}
			maps.Copy(tmp, event)
			tmp["message"] = randString(100)
			data = append(data, beat.Event{Timestamp: time.Now(), Fields: tmp})
		}

		out = append(out, eventInfo{data})
	}
	return out
}
