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
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/logp"
	beattest "github.com/elastic/beats/v7/libbeat/publisher/testing"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/format"
	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/json"
)

const (
	kafkaDefaultHost = "kafka"
	kafkaDefaultPort = "9092"
)

type testMessage struct {
	message string
	headers []sarama.RecordHeader
}

func recordHeader(key, value string) sarama.RecordHeader {
	return sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(value),
	}
}

func TestInput(t *testing.T) {
	testTopic := createTestTopicName()
	groupID := "filebeat"

	// Send test messages to the topic for the input to read.
	messages := []testMessage{
		testMessage{message: "testing"},
		testMessage{
			message: "stuff",
			headers: []sarama.RecordHeader{
				recordHeader("X-Test-Header", "test header value"),
			},
		},
		testMessage{
			message: "things",
			headers: []sarama.RecordHeader{
				recordHeader("keys and things", "3^3 = 27"),
				recordHeader("kafka yay", "3^3 - 2^4 = 11"),
			},
		},
	}
	for _, m := range messages {
		writeToKafkaTopic(t, testTopic, m.message, m.headers, time.Second*20)
	}

	// Setup the input config
	config := common.MustNewConfigFrom(common.MapStr{
		"hosts":      getTestKafkaHost(),
		"topics":     []string{testTopic},
		"group_id":   groupID,
		"wait_close": 5,
	})

	client := beattest.NewChanClient(100)
	defer client.Close()
	events := client.Channel
	input, cancel := run(t, config, client)

	timeout := time.After(30 * time.Second)
	for range messages {
		select {
		case event := <-events:
			v, err := event.Fields.GetValue("message")
			if err != nil {
				t.Fatal(err)
			}
			text, ok := v.(string)
			if !ok {
				t.Fatal("could not get message text from event")
			}
			msg := findMessage(t, text, messages)
			assert.Equal(t, text, msg.message)

			checkMatchingHeaders(t, event, msg.headers)

			// emulating the pipeline (kafkaInput.Run)
			meta, ok := event.Private.(eventMeta)
			if !ok {
				t.Fatal("could not get eventMeta and ack the message")
			}
			meta.ackHandler()
		case <-timeout:
			t.Fatal("timeout waiting for incoming events")
		}
	}

	// Close the done channel and make sure the beat shuts down in a reasonable
	// amount of time.
	cancel()
	didClose := make(chan struct{})
	go func() {
		input.Wait()
		close(didClose)
	}()

	select {
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for beat to shut down")
	case <-didClose:
	}

	assertOffset(t, groupID, testTopic, int64(len(messages)))
}

func TestInputWithMultipleEvents(t *testing.T) {
	testTopic := createTestTopicName()

	// Send test messages to the topic for the input to read.
	message := testMessage{
		message: "{\"records\": [{\"val\":\"val1\"}, {\"val\":\"val2\"}]}",
		headers: []sarama.RecordHeader{
			recordHeader("X-Test-Header", "test header value"),
		},
	}
	writeToKafkaTopic(t, testTopic, message.message, message.headers, time.Second*20)

	// Setup the input config
	config := common.MustNewConfigFrom(common.MapStr{
		"hosts":                        getTestKafkaHost(),
		"topics":                       []string{testTopic},
		"group_id":                     "filebeat",
		"wait_close":                   0,
		"expand_event_list_from_field": "records",
	})

	client := beattest.NewChanClient(100)
	defer client.Close()
	events := client.Channel
	input, cancel := run(t, config, client)

	timeout := time.After(30 * time.Second)
	select {
	case event := <-events:
		text, err := event.Fields.GetValue("message")
		if err != nil {
			t.Fatal(err)
		}
		msgs := []string{"{\"val\":\"val1\"}", "{\"val\":\"val2\"}"}
		assert.Contains(t, msgs, text)
		checkMatchingHeaders(t, event, message.headers)
	case <-timeout:
		t.Fatal("timeout waiting for incoming events")
	}

	cancel()
	// Close the done channel and make sure the beat shuts down in a reasonable
	// amount of time.
	didClose := make(chan struct{})
	go func() {
		input.Wait()
		close(didClose)
	}()

	select {
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for beat to shut down")
	case <-didClose:
	}
}

func TestInputWithJsonPayload(t *testing.T) {
	testTopic := createTestTopicName()

	// Send test message to the topic for the input to read.
	message := testMessage{
		message: "{\"val\":\"val1\"}",
		headers: []sarama.RecordHeader{
			recordHeader("X-Test-Header", "test header value"),
		},
	}
	writeToKafkaTopic(t, testTopic, message.message, message.headers, time.Second*20)

	// Setup the input config
	config := common.MustNewConfigFrom(common.MapStr{
		"hosts":      getTestKafkaHost(),
		"topics":     []string{testTopic},
		"group_id":   "filebeat",
		"wait_close": 0,
		"parsers": []common.MapStr{
			{
				"ndjson": common.MapStr{
					"target": "",
				},
			},
		},
	})

	client := beattest.NewChanClient(100)
	defer client.Close()
	events := client.Channel
	input, cancel := run(t, config, client)

	timeout := time.After(30 * time.Second)
	select {
	case event := <-events:
		text, err := event.Fields.GetValue("val")
		if err != nil {
			t.Fatal(err)
		}
		msgs := []string{"val1"}
		assert.Contains(t, msgs, text)
		checkMatchingHeaders(t, event, message.headers)
	case <-timeout:
		t.Fatal("timeout waiting for incoming events")
	}

	cancel()
	// Close the done channel and make sure the beat shuts down in a reasonable
	// amount of time.
	didClose := make(chan struct{})
	go func() {
		input.Wait()
		close(didClose)
	}()

	select {
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for beat to shut down")
	case <-didClose:
	}
}

func TestInputWithJsonPayloadAndMultipleEvents(t *testing.T) {
	testTopic := createTestTopicName()

	// Send test messages to the topic for the input to read.
	message := testMessage{
		message: "{\"records\": [{\"val\":\"val1\"}, {\"val\":\"val2\"}]}",
		headers: []sarama.RecordHeader{
			recordHeader("X-Test-Header", "test header value"),
		},
	}
	writeToKafkaTopic(t, testTopic, message.message, message.headers, time.Second*20)

	// Setup the input config
	config := common.MustNewConfigFrom(common.MapStr{
		"hosts":                        getTestKafkaHost(),
		"topics":                       []string{testTopic},
		"group_id":                     "filebeat",
		"wait_close":                   0,
		"expand_event_list_from_field": "records",
		"parsers": []common.MapStr{
			{
				"ndjson": common.MapStr{
					"target": "",
				},
			},
		},
	})

	client := beattest.NewChanClient(100)
	defer client.Close()
	events := client.Channel
	input, cancel := run(t, config, client)

	timeout := time.After(30 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case event := <-events:
			text, err := event.Fields.GetValue("val")
			if err != nil {
				t.Fatal(err)
			}
			msgs := []string{"val1", "val2"}
			assert.Contains(t, msgs, text)
			checkMatchingHeaders(t, event, message.headers)
		case <-timeout:
			t.Fatal("timeout waiting for incoming events")
		}
	}

	cancel()
	// Close the done channel and make sure the beat shuts down in a reasonable
	// amount of time.
	didClose := make(chan struct{})
	go func() {
		input.Wait()
		close(didClose)
	}()

	select {
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for beat to shut down")
	case <-didClose:
	}
}

func TestTest(t *testing.T) {
	testTopic := createTestTopicName()

	// Send test messages to the topic for the input to read.
	message := testMessage{
		message: "{\"records\": [{\"val\":\"val1\"}, {\"val\":\"val2\"}]}",
		headers: []sarama.RecordHeader{
			recordHeader("X-Test-Header", "test header value"),
		},
	}
	writeToKafkaTopic(t, testTopic, message.message, message.headers, time.Second*20)

	// Setup the input config
	config := common.MustNewConfigFrom(common.MapStr{
		"hosts":    getTestKafkaHost(),
		"topics":   []string{testTopic},
		"group_id": "filebeat",
	})

	inp, err := Plugin().Manager.Create(config)
	if err != nil {
		t.Fatal(err)
	}

	err = inp.Test(v2.TestContext{
		Logger: logp.NewLogger("kafka_test"),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func createTestTopicName() string {
	id := strconv.Itoa(rand.New(rand.NewSource(int64(time.Now().Nanosecond()))).Int())
	testTopic := fmt.Sprintf("Filebeat-TestInput-%s", id)
	return testTopic
}

func findMessage(t *testing.T, text string, msgs []testMessage) *testMessage {
	var msg *testMessage
	for _, m := range msgs {
		if text == m.message {
			msg = &m
			break
		}
	}

	assert.NotNil(t, msg)
	return msg
}

func checkMatchingHeaders(
	t *testing.T, event beat.Event, expected []sarama.RecordHeader,
) {
	kafka, err := event.Fields.GetValue("kafka")
	if err != nil {
		t.Fatal(err)
	}
	kafkaMap, ok := kafka.(common.MapStr)
	if !ok {
		t.Fatal("event.Fields.kafka isn't MapStr")
	}
	headers, err := kafkaMap.GetValue("headers")
	if err != nil {
		t.Fatal(err)
	}
	headerArray, ok := headers.([]string)
	if !ok {
		t.Fatal("event.Fields.kafka.headers isn't a []string")
	}
	assert.Equal(t, len(expected), len(headerArray))
	for i := 0; i < len(expected); i++ {
		splitIndex := strings.Index(headerArray[i], ": ")
		if splitIndex == -1 {
			t.Errorf(
				"event.Fields.kafka.headers[%v] doesn't have form 'key: value'", i)
			continue
		}
		key := headerArray[i][:splitIndex]
		value := headerArray[i][splitIndex+2:]
		assert.Equal(t, string(expected[i].Key), key)
		assert.Equal(t, string(expected[i].Value), value)
	}
}

func strDefault(a, defaults string) string {
	if a == "" {
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

func assertOffset(t *testing.T, groupID, topic string, expected int64) {
	client, err := sarama.NewClient([]string{getTestKafkaHost()}, nil)
	assert.NoError(t, err)
	defer client.Close()

	ofm, err := sarama.NewOffsetManagerFromClient(groupID, client)
	assert.NoError(t, err)
	defer ofm.Close()

	partitions, err := client.Partitions(topic)
	assert.NoError(t, err)

	var offsetSum int64

	for _, partitionID := range partitions {
		pom, err := ofm.ManagePartition(topic, partitionID)
		assert.NoError(t, err)

		offset, _ := pom.NextOffset()
		offsetSum += offset

		pom.Close()
	}

	assert.Equal(t, expected, offsetSum, "offset does not match, perhaps messages were not acknowledged")
}

func writeToKafkaTopic(
	t *testing.T, topic string, message string,
	headers []sarama.RecordHeader, timeout time.Duration,
) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	config.Producer.Partitioner = sarama.NewHashPartitioner
	config.Version = sarama.V1_0_0_0

	hosts := []string{getTestKafkaHost()}
	producer, err := sarama.NewSyncProducer(hosts, config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := producer.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	msg := &sarama.ProducerMessage{
		Topic:   topic,
		Value:   sarama.StringEncoder(message),
		Headers: headers,
	}

	_, _, err = producer.SendMessage(msg)
	if err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, cfg *common.Config, client *beattest.ChanClient) (*kafkaInput, func()) {
	inp, err := Plugin().Manager.Create(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := newV2Context()
	t.Cleanup(cancel)

	pipeline := beattest.ConstClient(client)
	input := inp.(*kafkaInput)
	go input.Run(ctx, pipeline)
	return input, cancel
}

func newV2Context() (v2.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	return v2.Context{
		Logger:      logp.NewLogger("kafka_test"),
		ID:          "test_id",
		Cancelation: ctx,
	}, cancel
}
