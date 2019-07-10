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

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
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
	events    chan *util.Data
}

func NewEventCapturer(events chan *util.Data) channel.Outleter {
	return &eventCapturer{
		c:      make(chan struct{}),
		events: events,
	}
}

func (o *eventCapturer) OnEvent(event *util.Data) bool {
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

func TestInput(t *testing.T) {
	id := strconv.Itoa(rand.New(rand.NewSource(int64(time.Now().Nanosecond()))).Int())
	testTopic := fmt.Sprintf("Filebeat-TestInput-%s", id)
	context := input.Context{
		Done:     make(chan struct{}),
		BeatDone: make(chan struct{}),
	}

	// Send test messages to the topic for the input to read.
	messageStrs := []string{"testing", "stuff", "blah"}
	for _, s := range messageStrs {
		writeToKafkaTopic(t, testTopic, s, time.Second*20)
	}

	// Setup the input config
	config, _ := common.NewConfigFrom(common.MapStr{
		"hosts":    "kafka:9092",
		"topics":   []string{testTopic},
		"group_id": "filebeat",
	})

	// Route input events through our capturer instead of sending through ES.
	events := make(chan *util.Data, 100)
	defer close(events)
	capturer := NewEventCapturer(events)
	defer capturer.Close()
	connector := func(*common.Config, *common.MapStrPointer) (channel.Outleter, error) {
		return channel.SubOutlet(capturer), nil
	}

	input, err := NewInput(config, connector, context)
	if err != nil {
		t.Fatal(err)
	}

	// Run the input and wait for finalization
	input.Run()

	timeout := time.After(30 * time.Second)
	for _, m := range messageStrs {
		select {
		case event := <-events:
			result, err := event.GetEvent().Fields.GetValue("message")
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, result, m)
		case <-timeout:
			t.Fatal("timeout waiting for incoming events")
		}
	}
}

func validateJSON(t *testing.T, value []byte, event beat.Event) {
	var decoded map[string]interface{}
	err := json.Unmarshal(value, &decoded)
	if err != nil {
		t.Errorf("can not json decode event value: %v", value)
		return
	}
	assert.Equal(t, decoded["type"], event.Fields["type"])
	assert.Equal(t, decoded["message"], event.Fields["message"])
}

func makeValidateFmtStr(fmt string) func(*testing.T, []byte, beat.Event) {
	fmtString := fmtstr.MustCompileEvent(fmt)
	return func(t *testing.T, value []byte, event beat.Event) {
		expectedMessage, err := fmtString.Run(&event)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, string(expectedMessage), string(value))
	}
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

func writeToKafkaTopic(
	t *testing.T, topic string, message string, timeout time.Duration,
) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	config.Producer.Partitioner = sarama.NewHashPartitioner

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
		Topic: topic,
		Value: sarama.StringEncoder(message),
	}

	_, _, err = producer.SendMessage(msg)
	if err != nil {
		t.Fatal(err)
	}
}
