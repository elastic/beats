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

package mqtt

import (
	"fmt"
	"sync"
	"testing"
	"time"

	libmqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/input/mqtt/testutil"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	message = "hello-world"

	waitTimeout = 30 * time.Second
)

var topic = fmt.Sprintf("topic-%d", time.Now().UnixNano())

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
	config := conf.MustNewConfigFrom(mapstr.M{
		"hosts":  []string{testutil.HostPort()},
		"topics": []string{topic},
	})

	eventsCh := make(chan beat.Event)
	defer close(eventsCh)

	captor := newEventCaptor(eventsCh)
	defer captor.Close()

	connector := channel.ConnectorFunc(func(_ *conf.C, _ beat.ClientConfig) (channel.Outleter, error) {
		return channel.SubOutlet(captor), nil
	})

	inputContext := input.Context{
		Done:     make(chan struct{}),
		BeatDone: make(chan struct{}),
	}

	logger := logptest.NewTestingLogger(t, "")
	input, err := NewInput(config, connector, inputContext, logger)
	require.NoError(t, err)
	require.NotNil(t, input)

	input.Run()

	publisher := testutil.CreatePublisher(t, "mqtt-integration-pub")

	verifiedCh := make(chan struct{})
	defer close(verifiedCh)

	emitInputData(t, verifiedCh, publisher)

	event := <-eventsCh
	verifiedCh <- struct{}{}

	val, err := event.GetValue("message")
	require.NoError(t, err)
	require.Equal(t, message, val)
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
				testutil.PublishMessage(t, publisher, topic, message)
			}
		}
	}()
}
