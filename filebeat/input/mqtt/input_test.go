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

	libmqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/require"

	finput "github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	logger = logp.NewLogger("test")
)

func TestNewInput_MissingConfigField(t *testing.T) {
	config := common.MustNewConfigFrom(common.MapStr{
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
	config := common.MustNewConfigFrom(common.MapStr{
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
	config := common.MustNewConfigFrom(common.MapStr{
		"hosts":  "tcp://mocked:1234",
		"topics": []string{"first", "second"},
		"qos":    2,
	})

	eventsCh := make(chan beat.Event)
	outlet := &mockedOutleter{
		events: eventsCh,
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
	newMqttClient = func(o *libmqtt.ClientOptions) libmqtt.Client {
		client = &mockedClient{
			onConnectHandler: o.OnConnect,
			messages:         []mockedMessage{firstMessage, secondMessage},
		}
		return client
	}

	input, err := NewInput(config, connector, inputContext)
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

func TestNewInput_Run_Stop(t *testing.T) {
	config := common.MustNewConfigFrom(common.MapStr{
		"hosts":  "tcp://mocked:1234",
		"topics": []string{"first", "second"},
		"qos":    2,
	})

	eventsCh := make(chan beat.Event)
	outlet := &mockedOutleter{
		events: eventsCh,
	}
	connector := &mockedConnector{
		outlet: outlet,
	}
	var inputContext finput.Context

	const numMessages = 5
	var messages []mockedMessage
	for i := 0; i < numMessages; i++ {
		messages = append(messages, mockedMessage{
			duplicate: false,
			messageID: 1,
			qos:       2,
			retained:  false,
			topic:     "first",
			payload:   []byte("first-message"),
		})
	}

	var client *mockedClient
	newMqttClient = func(o *libmqtt.ClientOptions) libmqtt.Client {
		client = &mockedClient{
			onConnectHandler: o.OnConnect,
			messages:         messages,
		}
		return client
	}

	input, err := NewInput(config, connector, inputContext)
	require.NoError(t, err)
	require.NotNil(t, input)

	input.Run()

	go func() {
		for range eventsCh {}
	}()

	input.Stop()
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

func TestStop(t *testing.T) {
	inflightMessages := new(sync.WaitGroup)
	client := new(mockedClient)
	input := &mqttInput{
		client:           client,
		logger:           logger,
		inflightMessages: inflightMessages,
	}

	input.Stop()

	require.Equal(t, 1, client.disconnectCount)
}

func TestWait(t *testing.T) {
	inflightMessages := new(sync.WaitGroup)
	input := &mqttInput{
		logger:           logger,
		inflightMessages: inflightMessages,
	}

	input.Wait()
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
