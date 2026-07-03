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

// This file was contributed to by generative AI
//go:build integration

package kafka

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/v7/filebeat/input/kafka/testutil"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	beattest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	"github.com/elastic/beats/v7/testing/testutils"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/sarama"

	"github.com/elastic/beats/v7/libbeat/beat"
	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/format"
	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/json"
)

type testMessage struct {
	message string
	headers []sarama.RecordHeader
}

func TestInput(t *testing.T) {
	testTopic := createReadyTestTopic(t)
	groupID := "filebeat"

	// Send test messages to the topic for the input to read.
	messages := []testMessage{
		{message: "testing"},
		{
			message: "stuff",
			headers: []sarama.RecordHeader{
				testutil.RecordHeader("X-Test-Header", "test header value"),
			},
		},
		{
			message: "things",
			headers: []sarama.RecordHeader{
				testutil.RecordHeader("keys and things", "3^3 = 27"),
				testutil.RecordHeader("kafka yay", "3^3 - 2^4 = 11"),
			},
		},
	}
	for _, m := range messages {
		testutil.WriteToKafkaTopic(t, testTopic, m.message, m.headers)
	}

	// Setup the input config
	config := conf.MustNewConfigFrom(mapstr.M{
		"hosts":      testutil.GetTestKafkaHost(),
		"topics":     []string{testTopic},
		"group_id":   groupID,
		"wait_close": 0,
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

	// sarama commits every second, we need to make sure
	// all message acks are committed before the rest of the checks
	<-time.After(2 * time.Second)

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
	testTopic := createReadyTestTopic(t)

	// Send test messages to the topic for the input to read.
	message := testMessage{
		message: "{\"records\": [{\"val\":\"val1\"}, {\"val\":\"val2\"}]}",
		headers: []sarama.RecordHeader{
			testutil.RecordHeader("X-Test-Header", "test header value"),
		},
	}
	testutil.WriteToKafkaTopic(t, testTopic, message.message, message.headers)

	// Setup the input config
	config := conf.MustNewConfigFrom(mapstr.M{
		"hosts":                        testutil.GetTestKafkaHost(),
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
	testTopic := createReadyTestTopic(t)

	// Send test message to the topic for the input to read.
	message := testMessage{
		message: "{\"val\":\"val1\"}",
		headers: []sarama.RecordHeader{
			testutil.RecordHeader("X-Test-Header", "test header value"),
		},
	}
	testutil.WriteToKafkaTopic(t, testTopic, message.message, message.headers)

	// Setup the input config
	config := conf.MustNewConfigFrom(mapstr.M{
		"hosts":      testutil.GetTestKafkaHost(),
		"topics":     []string{testTopic},
		"group_id":   "filebeat",
		"wait_close": 0,
		"parsers": []mapstr.M{
			{
				"ndjson": mapstr.M{
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
	testTopic := createReadyTestTopic(t)

	// Send test messages to the topic for the input to read.
	message := testMessage{
		message: "{\"records\": [{\"val\":\"val1\"}, {\"val\":\"val2\"}]}",
		headers: []sarama.RecordHeader{
			testutil.RecordHeader("X-Test-Header", "test header value"),
		},
	}
	testutil.WriteToKafkaTopic(t, testTopic, message.message, message.headers)

	// Setup the input config
	config := conf.MustNewConfigFrom(mapstr.M{
		"hosts":                        testutil.GetTestKafkaHost(),
		"topics":                       []string{testTopic},
		"group_id":                     "filebeat",
		"wait_close":                   0,
		"expand_event_list_from_field": "records",
		"parsers": []mapstr.M{
			{
				"ndjson": mapstr.M{
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

func TestSASLAuthentication(t *testing.T) {
	testutils.SkipIfFIPSOnly(t, "SASL disabled when in fips140=only mode.")

	testCases := []struct {
		name      string
		mechanism string
	}{
		{
			name:      "SCRAM-SHA-256",
			mechanism: sarama.SASLTypeSCRAMSHA256,
		},
		{
			name:      "SCRAM-SHA-512",
			mechanism: sarama.SASLTypeSCRAMSHA512,
		},
		{
			name:      "PLAIN",
			mechanism: sarama.SASLTypePlaintext,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testTopic := createReadyTestTopic(t)
			groupID := "filebeat"

			// Send test messages to the topic for the input to read.
			messages := []testMessage{
				{message: "testing"},
				{message: fmt.Sprintf("sasl test with %s", tc.name)},
			}
			for _, m := range messages {
				testutil.WriteToKafkaTopic(t, testTopic, m.message, m.headers)
			}

			// Setup the input config
			config := conf.MustNewConfigFrom(mapstr.M{
				"hosts":          []string{testutil.GetTestSASLKafkaHost()},
				"protocol":       "https",
				"sasl.mechanism": tc.mechanism,
				// Disable hostname verification since we are likely writing to localhost.
				"ssl.verification_mode": "certificate",
				"ssl.certificate_authorities": []string{
					"../../../testing/environments/docker/kafka/certs/ca-cert",
				},
				"username": "beats",
				"password": "KafkaTest",

				"topics":     []string{testTopic},
				"group_id":   groupID,
				"wait_close": 0,
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

			// sarama commits every second, we need to make sure
			// all message acks are committed before the rest of the checks
			<-time.After(2 * time.Second)

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
		})
	}
}

func TestTest(t *testing.T) {
	testTopic := createReadyTestTopic(t)

	// Send test messages to the topic for the input to read.
	message := testMessage{
		message: "{\"records\": [{\"val\":\"val1\"}, {\"val\":\"val2\"}]}",
		headers: []sarama.RecordHeader{
			testutil.RecordHeader("X-Test-Header", "test header value"),
		},
	}
	testutil.WriteToKafkaTopic(t, testTopic, message.message, message.headers)

	// Setup the input config
	config := conf.MustNewConfigFrom(mapstr.M{
		"hosts":    testutil.GetTestKafkaHost(),
		"topics":   []string{testTopic},
		"group_id": "filebeat",
	})

	inp, err := Plugin(logptest.NewTestingLogger(t, "")).Manager.Create(config)
	if err != nil {
		t.Fatal(err)
	}

	err = inp.Test(v2.TestContext{
		Logger: logptest.NewTestingLogger(t, "kafka_test"),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func createReadyTestTopic(t *testing.T) string {
	t.Helper()

	testTopic := fmt.Sprintf("Filebeat-TestInput-%d", rand.Int())
	// Topic auto-creation is asynchronous; explicitly wait for leaders to avoid
	// transient "no leader for this partition" write failures in CI.
	testutil.EnsureKafkaTopicReadyForWrites(t, testTopic)
	return testTopic
}

func findMessage(t *testing.T, text string, msgs []testMessage) *testMessage {
	var msg *testMessage
	for _, m := range msgs {
		if text == m.message {
			mCopy := m
			msg = &mCopy
			break
		}
	}

	assert.NotNil(t, msg)
	return msg
}

func checkMatchingHeaders(
	t *testing.T, event beat.Event, expected []sarama.RecordHeader,
) {
	t.Helper()
	kafka, err := event.Fields.GetValue("kafka")
	if err != nil {
		t.Fatal(err)
	}
	kafkaMap, ok := kafka.(mapstr.M)
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
	assert.Len(t, headerArray, len(expected))
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

func assertOffset(t *testing.T, groupID, topic string, expected int64) {
	t.Helper()
	client, err := sarama.NewClient([]string{testutil.GetTestKafkaHost()}, nil)
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
		// if the partition was not written to before
		// it might return -1 which would affect the sum
		if offset > 0 {
			offsetSum += offset
		}

		pom.Close()
	}

	assert.Equal(t, expected, offsetSum, "offset does not match, perhaps messages were not acknowledged")
}

func run(t *testing.T, cfg *conf.C, client *beattest.ChanClient) (*kafkaInput, func()) {
	inp, err := Plugin(logptest.NewTestingLogger(t, "")).Manager.Create(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := newV2Context()
	t.Cleanup(cancel)

	pipeline := beattest.ConstClient(client)
	input, _ := inp.(*kafkaInput)
	go func() {
		_ = input.Run(ctx, pipeline)
	}()
	return input, cancel
}

func newV2Context() (v2.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	logger, _ := logp.NewDevelopmentLogger("kafka_test")
	return v2.Context{
		Logger:      logger,
		ID:          "test_id",
		Cancelation: ctx,
	}, cancel
}
