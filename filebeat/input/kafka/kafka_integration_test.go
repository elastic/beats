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

// +build integration

package kafka

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	_ "github.com/elastic/beats/libbeat/outputs/codec/format"
	_ "github.com/elastic/beats/libbeat/outputs/codec/json"
)

const (
	kafkaDefaultHost = "kafka"
	kafkaDefaultPort = "9092"
)

type eventInfo struct {
	events []beat.Event
}

type eventCapturer struct {
	closed    bool
	c         chan struct{}
	closeOnce sync.Once
	events    chan beat.Event
}

func NewEventCapturer(events chan beat.Event) channel.Outleter {
	return &eventCapturer{
		c:      make(chan struct{}),
		events: events,
	}
}

func (o *eventCapturer) OnEvent(event beat.Event) bool {
	o.events <- event
	return true
}

func (o *eventCapturer) Close() error {
	o.closeOnce.Do(func() {
		o.closed = true
		close(o.c)
	})
	return nil
}

func (o *eventCapturer) Done() <-chan struct{} {
	return o.c
}

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
	id := strconv.Itoa(rand.New(rand.NewSource(int64(time.Now().Nanosecond()))).Int())
	testTopic := fmt.Sprintf("Filebeat-TestInput-%s", id)
	context := input.Context{
		Done:     make(chan struct{}),
		BeatDone: make(chan struct{}),
	}

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
		"group_id":   "filebeat",
		"wait_close": 0,
	})

	// Route input events through our capturer instead of sending through ES.
	events := make(chan beat.Event, 100)
	defer close(events)
	capturer := NewEventCapturer(events)
	defer capturer.Close()
	connector := channel.ConnectorFunc(func(_ *common.Config, _ beat.ClientConfig) (channel.Outleter, error) {
		return channel.SubOutlet(capturer), nil
	})

	input, err := NewInput(config, connector, context)
	if err != nil {
		t.Fatal(err)
	}

	// Run the input and wait for finalization
	input.Run()

	timeout := time.After(30 * time.Second)
	for _, m := range messages {
		select {
		case event := <-events:
			text, err := event.Fields.GetValue("message")
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, text, m.message)

			checkMatchingHeaders(t, event, m.headers)
		case <-timeout:
			t.Fatal("timeout waiting for incoming events")
		}
	}

	// Close the done channel and make sure the beat shuts down in a reasonable
	// amount of time.
	close(context.Done)
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

func TestInputWithMultipleEvents(t *testing.T) {
	id := strconv.Itoa(rand.New(rand.NewSource(int64(time.Now().Nanosecond()))).Int())
	testTopic := fmt.Sprintf("Filebeat-TestInput-%s", id)
	context := input.Context{
		Done:     make(chan struct{}),
		BeatDone: make(chan struct{}),
	}

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

	// Route input events through our capturer instead of sending through ES.
	events := make(chan beat.Event, 100)
	defer close(events)
	capturer := NewEventCapturer(events)
	defer capturer.Close()
	connector := channel.ConnectorFunc(func(_ *common.Config, _ beat.ClientConfig) (channel.Outleter, error) {
		return channel.SubOutlet(capturer), nil
	})

	input, err := NewInput(config, connector, context)
	if err != nil {
		t.Fatal(err)
	}

	// Run the input and wait for finalization
	input.Run()

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

	// Close the done channel and make sure the beat shuts down in a reasonable
	// amount of time.
	close(context.Done)
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

func checkMatchingHeaders(
	t *testing.T, event beat.Event, expected []sarama.RecordHeader,
) {
	kafka, err := event.Fields.GetValue("kafka")
	if err != nil {
		t.Error(err)
		return
	}
	kafkaMap, ok := kafka.(common.MapStr)
	if !ok {
		t.Error("event.Fields.kafka isn't MapStr")
		return
	}
	headers, err := kafkaMap.GetValue("headers")
	if err != nil {
		t.Error(err)
		return
	}
	headerArray, ok := headers.([]string)
	if !ok {
		t.Error("event.Fields.kafka.headers isn't a []string")
		return
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
