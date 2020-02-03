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
	"strings"
	"sync"
	"time"

	libmqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/backoff"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	disconnectTimeout = 3 * 1000 // 3000 ms = 3 sec

	subscribeTimeout       = 35 * time.Second // in client: subscribeWaitTimeout = 30s
	subscribeRetryInterval = 1 * time.Second
)

// Input contains the input and its config
type mqttInput struct {
	once sync.Once

	logger *logp.Logger

	client           libmqtt.Client
	inflightMessages *sync.WaitGroup
}

func init() {
	err := input.Register("mqtt", NewInput)
	if err != nil {
		panic(err)
	}
}

// NewInput method creates a new mqtt input,
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	inputContext input.Context,
) (input.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "reading mqtt input config")
	}

	out, err := connector.ConnectWith(cfg, beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			DynamicFields: inputContext.DynamicFields,
		},
	})
	if err != nil {
		return nil, err
	}

	logger := logp.NewLogger("mqtt input").With("hosts", config.Hosts)
	setupLibraryLogging()

	inflightMessages := new(sync.WaitGroup)
	clientSubscriptions := createClientSubscriptions(config)
	onMessageHandler := createOnMessageHandler(logger, out, inflightMessages)
	onConnectHandler := createOnConnectHandler(logger, &inputContext, onMessageHandler, clientSubscriptions)
	clientOptions, err := createClientOptions(config, onConnectHandler)
	if err != nil {
		return nil, err
	}

	return &mqttInput{
		client:           libmqtt.NewClient(clientOptions),
		inflightMessages: inflightMessages,
		logger:           logp.NewLogger("mqtt input").With("hosts", config.Hosts),
	}, nil
}

func createOnMessageHandler(logger *logp.Logger, outlet channel.Outleter, inflightMessages *sync.WaitGroup) func(client libmqtt.Client, message libmqtt.Message) {
	return func(client libmqtt.Client, message libmqtt.Message) {
		inflightMessages.Add(1)

		logger.Debugf("Received message on topic '%s', messageID: %d, size: %d", message.Topic(),
			message.MessageID(), len(message.Payload()))

		mqttFields := common.MapStr{
			"duplicate":  message.Duplicate(),
			"message_id": message.MessageID(),
			"qos":        message.Qos(),
			"retained":   message.Retained(),
			"topic":      message.Topic(),
		}
		outlet.OnEvent(beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"message": string(message.Payload()),
				"mqtt":    mqttFields,
			},
		})

		inflightMessages.Done()
	}
}

func createOnConnectHandler(logger *logp.Logger, inputContext *input.Context, onMessageHandler func(client libmqtt.Client, message libmqtt.Message), clientSubscriptions map[string]byte) func(client libmqtt.Client) {
	// The function subscribes the client to the specific topics (with retry backoff in case of failure).
	return func(client libmqtt.Client) {
		backoff := backoff.NewEqualJitterBackoff(
			inputContext.Done,
			subscribeRetryInterval,
			8*subscribeRetryInterval)

		var topics []string
		for topic := range clientSubscriptions {
			topics = append(topics, topic)
		}

		var success bool
		for !success {
			logger.Debugf("Try subscribe to topics: %v", strings.Join(topics, ", "))

			token := client.SubscribeMultiple(clientSubscriptions, onMessageHandler)
			if !token.WaitTimeout(subscribeTimeout) {
				if token.Error() != nil {
					logger.Warnf("Subscribing to topics failed due to error: %v", token.Error())
				}

				if !backoff.Wait() {
					backoff.Reset()
					success = true
				}
			} else {
				backoff.Reset()
				success = true
			}
		}
	}
}

// Run method starts the mqtt input and processing.
// The mqtt client starts in auto-connect mode (with connection retries and resuming topic subscriptions).
func (mi *mqttInput) Run() {
	mi.once.Do(func() {
		mi.logger.Debug("Run the input once.")
		mi.client.Connect()
	})
}

// Stop method stops the input.
func (mi *mqttInput) Stop() {
	mi.logger.Debug("Stop the input.")
	mi.client.Disconnect(disconnectTimeout)
	mi.Wait()
}

// Wait method waits until event processing is finished.
func (mi *mqttInput) Wait() {
	mi.logger.Debug("Wait for the input to finish processing.")
	mi.inflightMessages.Wait()
}
