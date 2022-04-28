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

package mqtt

import (
	"errors"
	"sync"
	"testing"
	"time"

	libmqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/require"

	finput "github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/input/inputtest"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var logger = logp.NewLogger("test")

func TestNewInput_MissingConfigField(t *testing.T) {
	config := common.MustNewConfigFrom(mapstr.M{
		"topics": "#",
	})
	connector := new(mockedConnector)
	var inputContext finput.Context

	input, err := NewInput(config, connector, inputContext)

	require.Error(t, err)
	require.Nil(t, input)
}

func TestNewInput_ConnectWithFailed(t *testing.T) {
	connectWithError := errors.New("failure")
	config := common.MustNewConfigFrom(mapstr.M{
		"hosts":  "tcp://mocked:1234",
		"topics": "#",
	})
	connector := &mockedConnector{
		connectWithError: connectWithError,
	}
	var inputContext finput.Context

	input, err := NewInput(config, connector, inputContext)

	require.Equal(t, connectWithError, err)
	require.Nil(t, input)
}

func TestNewInput_Run(t *testing.T) {
	config := common.MustNewConfigFrom(mapstr.M{
		"hosts":  "tcp://mocked:1234",
		"topics": []string{"first", "second"},
		"qos":    2,
	})

	eventsCh := make(chan beat.Event)
	defer close(eventsCh)

	outlet := &mockedOutleter{
		onEventHandler: func(event beat.Event) bool {
			eventsCh <- event
			return true
		},
	}
	connector := &mockedConnector{
		outlet: outlet,
	}
	var inputContext finput.Context

	firstMessage := mockedMessage{
		duplicate: false,
		messageID: 1,
		qos:       2,
		retained:  false,
		topic:     "first",
		payload:   []byte("first-message"),
	}
	secondMessage := mockedMessage{
		duplicate: false,
		messageID: 2,
		qos:       2,
		retained:  false,
		topic:     "second",
		payload:   []byte("second-message"),
	}

	var client *mockedClient
	newMqttClient := func(o *libmqtt.ClientOptions) libmqtt.Client {
		client = &mockedClient{
			onConnectHandler: o.OnConnect,
			messages:         []mockedMessage{firstMessage, secondMessage},
			tokens: []libmqtt.Token{&mockedToken{
				timeout: true,
			}},
		}
		return client
	}

	input, err := newInput(config, connector, inputContext, newMqttClient, backoff.NewEqualJitterBackoff)
	require.NoError(t, err)
	require.NotNil(t, input)

	input.Run()

	require.Equal(t, 1, client.connectCount)
	require.Equal(t, 1, client.subscribeMultipleCount)
	require.ElementsMatch(t, []string{"first", "second"}, client.subscriptions)

	for _, event := range []beat.Event{<-eventsCh, <-eventsCh} {
		topic, err := event.GetValue("mqtt.topic")
		require.NoError(t, err)

		if topic == "first" {
			assertEventMatches(t, firstMessage, event)
		} else {
			assertEventMatches(t, secondMessage, event)
		}
	}
}

func TestNewInput_Run_Wait(t *testing.T) {
	config := common.MustNewConfigFrom(mapstr.M{
		"hosts":  "tcp://mocked:1234",
		"topics": []string{"first", "second"},
		"qos":    2,
	})

	const numMessages = 5

	var eventProcessing sync.WaitGroup
	eventProcessing.Add(numMessages)

	eventsCh := make(chan beat.Event)
	defer close(eventsCh)

	outlet := &mockedOutleter{
		onEventHandler: func(event beat.Event) bool {
			eventProcessing.Done()
			eventsCh <- event
			return true
		},
	}
	connector := &mockedConnector{
		outlet: outlet,
	}
	var inputContext finput.Context

	var messages []mockedMessage
	for i := 0; i < numMessages; i++ {
		messages = append(messages, mockedMessage{
			duplicate: false,
			messageID: 1,
			qos:       2,
			retained:  false,
			topic:     "topic",
			payload:   []byte("a-message"),
		})
	}

	var client *mockedClient
	newMqttClient := func(o *libmqtt.ClientOptions) libmqtt.Client {
		client = &mockedClient{
			onConnectHandler: o.OnConnect,
			messages:         messages,
			tokens: []libmqtt.Token{&mockedToken{
				timeout: true,
			}},
		}
		return client
	}

	input, err := newInput(config, connector, inputContext, newMqttClient, backoff.NewEqualJitterBackoff)
	require.NoError(t, err)
	require.NotNil(t, input)

	input.Run()
	eventProcessing.Wait()

	go func() {
		time.Sleep(100 * time.Millisecond) // let input.Stop() be executed.
		for range eventsCh {
		}
	}()

	input.Wait()
}

func TestRun_Once(t *testing.T) {
	client := new(mockedClient)
	input := &mqttInput{
		client: client,
		logger: logger,
	}

	input.Run()

	require.Equal(t, 1, client.connectCount)
}

func TestRun_Twice(t *testing.T) {
	client := new(mockedClient)
	input := &mqttInput{
		client: client,
		logger: logger,
	}

	input.Run()
	input.Run()

	require.Equal(t, 1, client.connectCount)
}

func TestWait(t *testing.T) {
	clientDisconnected := new(sync.WaitGroup)
	inflightMessages := new(sync.WaitGroup)
	client := new(mockedClient)
	input := &mqttInput{
		client:             client,
		clientDisconnected: clientDisconnected,
		logger:             logger,
		inflightMessages:   inflightMessages,
	}

	input.Wait()

	require.Equal(t, 1, client.disconnectCount)
}

func TestStop(t *testing.T) {
	client := new(mockedClient)
	clientDisconnected := new(sync.WaitGroup)
	input := &mqttInput{
		client:             client,
		clientDisconnected: clientDisconnected,
		logger:             logger,
	}

	input.Stop()
}

func TestOnCreateHandler_SubscribeMultiple_Succeeded(t *testing.T) {
	inputContext := new(finput.Context)
	onMessageHandler := func(client libmqtt.Client, message libmqtt.Message) {}
	var clientSubscriptions map[string]byte
	newBackoff := func(done <-chan struct{}, init, max time.Duration) backoff.Backoff {
		return backoff.NewEqualJitterBackoff(inputContext.Done, time.Nanosecond, 2*time.Nanosecond)
	}
	handler := createOnConnectHandler(logger, inputContext, onMessageHandler, clientSubscriptions, newBackoff)

	client := &mockedClient{
		tokens: []libmqtt.Token{&mockedToken{
			timeout: true,
		}},
	}
	handler(client)

	require.Equal(t, 1, client.subscribeMultipleCount)
}

func TestOnCreateHandler_SubscribeMultiple_BackoffSucceeded(t *testing.T) {
	inputContext := new(finput.Context)
	onMessageHandler := func(client libmqtt.Client, message libmqtt.Message) {}
	var clientSubscriptions map[string]byte
	newBackoff := func(done <-chan struct{}, init, max time.Duration) backoff.Backoff {
		return backoff.NewEqualJitterBackoff(inputContext.Done, time.Nanosecond, 2*time.Nanosecond)
	}
	handler := createOnConnectHandler(logger, inputContext, onMessageHandler, clientSubscriptions, newBackoff)

	client := &mockedClient{
		tokens: []libmqtt.Token{&mockedToken{
			timeout: false,
		}, &mockedToken{
			timeout: true,
		}},
	}
	handler(client)

	require.Equal(t, 2, client.subscribeMultipleCount)
}

func TestOnCreateHandler_SubscribeMultiple_BackoffSignalDone(t *testing.T) {
	inputContext := new(finput.Context)
	onMessageHandler := func(client libmqtt.Client, message libmqtt.Message) {}
	var clientSubscriptions map[string]byte
	mockedBackoff := &mockedBackoff{
		waits: []bool{true, false},
	}
	newBackoff := func(done <-chan struct{}, init, max time.Duration) backoff.Backoff {
		return mockedBackoff
	}
	handler := createOnConnectHandler(logger, inputContext, onMessageHandler, clientSubscriptions, newBackoff)

	client := &mockedClient{
		tokens: []libmqtt.Token{&mockedToken{
			timeout: false,
		}, &mockedToken{
			timeout: false,
		}},
	}
	handler(client)

	require.Equal(t, 2, client.subscribeMultipleCount)
	require.Equal(t, 1, mockedBackoff.resetCount)
}

func TestNewInputDone(t *testing.T) {
	config := mapstr.M{
		"hosts": "tcp://:0",
	}
	inputtest.AssertNotStartedInputCanBeDone(t, NewInput, &config)
}

func assertEventMatches(t *testing.T, expected mockedMessage, got beat.Event) {
	topic, err := got.GetValue("mqtt.topic")
	require.NoError(t, err)
	require.Equal(t, expected.topic, topic)

	duplicate, err := got.GetValue("mqtt.duplicate")
	require.NoError(t, err)
	require.Equal(t, expected.duplicate, duplicate)

	messageID, err := got.GetValue("mqtt.message_id")
	require.NoError(t, err)
	require.Equal(t, expected.messageID, messageID)

	qos, err := got.GetValue("mqtt.qos")
	require.NoError(t, err)
	require.Equal(t, expected.qos, qos)

	retained, err := got.GetValue("mqtt.retained")
	require.NoError(t, err)
	require.Equal(t, expected.retained, retained)

	message, err := got.GetValue("message")
	require.NoError(t, err)
	require.Equal(t, string(expected.payload), message)
}
