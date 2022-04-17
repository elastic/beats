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

package mqtt

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	libmqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/filebeat/channel"
	"github.com/menderesk/beats/v7/filebeat/input"
	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

const (
	message = "hello-world"

	waitTimeout = 30 * time.Second
)

var (
	hostPort = fmt.Sprintf("tcp://%s:%s",
		getOrDefault(os.Getenv("MOSQUITTO_HOST"), "mosquitto"),
		getOrDefault(os.Getenv("MOSQUITTO_PORT"), "1883"))
	topic = fmt.Sprintf("topic-%d", time.Now().UnixNano())
)

type eventCaptor struct {
	c         chan struct{}
	closeOnce sync.Once
	closed    bool
	events    chan beat.Event
}

func newEventCaptor(events chan beat.Event) channel.Outleter {
	return &eventCaptor{
		c:      make(chan struct{}),
		events: events,
	}
}

func (ec *eventCaptor) OnEvent(event beat.Event) bool {
	ec.events <- event
	return true
}

func (ec *eventCaptor) Close() error {
	ec.closeOnce.Do(func() {
		ec.closed = true
		close(ec.c)
	})
	return nil
}

func (ec *eventCaptor) Done() <-chan struct{} {
	return ec.c
}

func TestInput(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("mqtt input", "libmqtt"))

	// Setup the input config.
	config := common.MustNewConfigFrom(common.MapStr{
		"hosts":  []string{hostPort},
		"topics": []string{topic},
	})

	// Route input events through our captor instead of sending through ES.
	eventsCh := make(chan beat.Event)
	defer close(eventsCh)

	captor := newEventCaptor(eventsCh)
	defer captor.Close()

	connector := channel.ConnectorFunc(func(_ *common.Config, _ beat.ClientConfig) (channel.Outleter, error) {
		return channel.SubOutlet(captor), nil
	})

	// Mock the context.
	inputContext := input.Context{
		Done:     make(chan struct{}),
		BeatDone: make(chan struct{}),
	}

	// Setup the input
	input, err := NewInput(config, connector, inputContext)
	require.NoError(t, err)
	require.NotNil(t, input)

	// Run the input.
	input.Run()

	// Create Publisher
	publisher := createPublisher(t)

	// Verify that event has been received
	verifiedCh := make(chan struct{})
	defer close(verifiedCh)

	emitInputData(t, verifiedCh, publisher)

	event := <-eventsCh
	verifiedCh <- struct{}{}

	val, err := event.GetValue("message")
	require.NoError(t, err)
	require.Equal(t, message, val)
}

func createPublisher(t *testing.T) libmqtt.Client {
	clientOptions := libmqtt.NewClientOptions().
		SetClientID("emitter").
		SetAutoReconnect(false).
		SetConnectRetry(false).
		AddBroker(hostPort)
	client := libmqtt.NewClient(clientOptions)
	token := client.Connect()
	require.True(t, token.WaitTimeout(waitTimeout))
	require.NoError(t, token.Error())
	return client
}

func emitInputData(t *testing.T, verifiedCh <-chan struct{}, publisher libmqtt.Client) {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-verifiedCh:
				return
			case <-ticker.C:
				token := publisher.Publish(topic, 1, false, []byte(message))
				require.True(t, token.WaitTimeout(waitTimeout))
				require.NoError(t, token.Error())
			}
		}
	}()
}

func getOrDefault(s, defaultString string) string {
	if s == "" {
		return defaultString
	}
	return s
}
