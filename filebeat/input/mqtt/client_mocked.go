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
	"time"

	libmqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	conf "github.com/elastic/elastic-agent-libs/config"
)

type mockedMessage struct {
	duplicate bool
	messageID uint16
	qos       byte
	retained  bool
	topic     string
	payload   []byte
}

var _ libmqtt.Message = new(mockedMessage)

func (m *mockedMessage) Duplicate() bool {
	return m.duplicate
}

func (m *mockedMessage) Qos() byte {
	return m.qos
}

func (m *mockedMessage) Retained() bool {
	return m.retained
}

func (m *mockedMessage) Topic() string {
	return m.topic
}

func (m *mockedMessage) MessageID() uint16 {
	return m.messageID
}

func (m *mockedMessage) Payload() []byte {
	return m.payload
}

func (m *mockedMessage) Ack() {
	panic("implement me")
}

type mockedBackoff struct {
	resetCount int

	waits     []bool
	waitIndex int
}

var _ backoff.Backoff = new(mockedBackoff)

func (m *mockedBackoff) Wait() bool {
	wait := m.waits[m.waitIndex]
	m.waitIndex++
	return wait
}

func (m *mockedBackoff) Reset() {
	m.resetCount++
}

type mockedToken struct {
	timeout bool
}

var _ libmqtt.Token = new(mockedToken)

func (m *mockedToken) Wait() bool {
	panic("implement me")
}

func (m *mockedToken) WaitTimeout(time.Duration) bool {
	return m.timeout
}

func (m *mockedToken) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (m *mockedToken) Error() error {
	return nil
}

type mockedClient struct {
	connectCount           int
	disconnectCount        int
	subscribeMultipleCount int

	subscriptions []string
	messages      []mockedMessage

	tokens     []libmqtt.Token
	tokenIndex int

	onConnectHandler func(client libmqtt.Client)
	onMessageHandler func(client libmqtt.Client, message libmqtt.Message)
}

var _ libmqtt.Client = new(mockedClient)

func (m *mockedClient) IsConnected() bool {
	panic("implement me")
}

func (m *mockedClient) IsConnectionOpen() bool {
	panic("implement me")
}

func (m *mockedClient) Connect() libmqtt.Token {
	m.connectCount++

	if m.onConnectHandler != nil {
		m.onConnectHandler(m)
	}
	return nil
}

func (m *mockedClient) Disconnect(quiesce uint) {
	m.disconnectCount++
}

func (m *mockedClient) Publish(topic string, qos byte, retained bool, payload interface{}) libmqtt.Token {
	panic("implement me")
}

func (m *mockedClient) Subscribe(topic string, qos byte, callback libmqtt.MessageHandler) libmqtt.Token {
	panic("implement me")
}

func (m *mockedClient) SubscribeMultiple(filters map[string]byte, callback libmqtt.MessageHandler) libmqtt.Token {
	m.subscribeMultipleCount++

	for filter := range filters {
		m.subscriptions = append(m.subscriptions, filter)
	}
	m.onMessageHandler = callback

	for _, msg := range m.messages {
		thatMsg := msg
		go func() {
			m.onMessageHandler(m, &thatMsg)
		}()
	}

	token := m.tokens[m.tokenIndex]
	m.tokenIndex++
	return token
}

func (m *mockedClient) Unsubscribe(topics ...string) libmqtt.Token {
	panic("implement me")
}

func (m *mockedClient) AddRoute(topic string, callback libmqtt.MessageHandler) {
	panic("implement me")
}

func (m *mockedClient) OptionsReader() libmqtt.ClientOptionsReader {
	panic("implement me")
}

type mockedConnector struct {
	connectWithError error
	outlet           channel.Outleter
}

var _ channel.Connector = new(mockedConnector)

func (m *mockedConnector) Connect(c *conf.C) (channel.Outleter, error) {
	return m.ConnectWith(c, beat.ClientConfig{})
}

func (m *mockedConnector) ConnectWith(*conf.C, beat.ClientConfig) (channel.Outleter, error) {
	if m.connectWithError != nil {
		return nil, m.connectWithError
	}
	return m.outlet, nil
}

type mockedOutleter struct {
	onEventHandler func(event beat.Event) bool
}

var _ channel.Outleter = new(mockedOutleter)

func (m mockedOutleter) Close() error {
	panic("implement me")
}

func (m mockedOutleter) Done() <-chan struct{} {
	panic("implement me")
}

func (m mockedOutleter) OnEvent(event beat.Event) bool {
	return m.onEventHandler(event)
}
