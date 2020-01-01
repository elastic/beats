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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"gopkg.in/vmihailenco/msgpack.v2"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func (input *mqttInput) newTLSConfig() *tls.Config {
	config := input.config

	// Import trusted certificates from CAfile.pem.
	// Alternatively, manually add CA certificates to
	// default openssl CA bundle.
	certpool := x509.NewCertPool()
	if config.CA != "" {
		logp.Info("[MQTT] Set the CA")
		pemCerts, err := ioutil.ReadFile(config.CA)
		if err == nil {
			certpool.AppendCertsFromPEM(pemCerts)
		}
	}

	tlsconfig := &tls.Config{
		// RootCAs = certs used to verify server cert.
		RootCAs: certpool,
		// ClientAuth = whether to request cert from server.
		// Since the server is set up for SSL, this happens
		// anyways.
		ClientAuth: tls.NoClientCert,
		// ClientCAs = certs used to validate client cert.
		ClientCAs: nil,
		// InsecureSkipVerify = verify that cert contents
		// match server. IP matches what is in cert etc.
		InsecureSkipVerify: true,
	}

	// Import client certificate/key pair
	if config.ClientCert != "" && config.ClientKey != "" {
		logp.Info("[MQTT] Set the Certs")
		cert, err := tls.LoadX509KeyPair(config.ClientCert, config.ClientKey)
		if err != nil {
			panic(err)
		}

		// Certificates = list of certs client sends to server.
		tlsconfig.Certificates = []tls.Certificate{cert}
	}

	// Create tls.Config with desired tls properties
	return tlsconfig
}

// Prepare MQTT client
func (input *mqttInput) setupMqttClient() {
	c := input.config

	logp.Info("[MQTT] Connect to broker URL: %s", c.Host)

	mqttClientOpt := MQTT.NewClientOptions()
	mqttClientOpt.SetClientID(c.ClientID)
	mqttClientOpt.AddBroker(c.Host)

	mqttClientOpt.SetMaxReconnectInterval(1 * time.Second)
	mqttClientOpt.SetConnectionLostHandler(input.reConnectHandler)
	mqttClientOpt.SetOnConnectHandler(input.subscribeOnConnect)
	mqttClientOpt.SetAutoReconnect(true)

	if c.Username != "" {
		logp.Info("[MQTT] Broker username: %s", c.Username)
		mqttClientOpt.SetUsername(c.Username)
	}

	if c.Password != "" {
		mqttClientOpt.SetPassword(c.Password)
	}

	if c.SSL == true {
		logp.Info("[MQTT] Configure session to use SSL")
		tlsconfig := input.newTLSConfig()
		mqttClientOpt.SetTLSConfig(tlsconfig)
	}

	input.client = MQTT.NewClient(mqttClientOpt)
}

func (input *mqttInput) connect() error {
	if token := input.client.Connect(); token.Wait() && token.Error() != nil {
		return errors.New("Failed to connect to broker, waiting a few seconds and retrying")
	}
	logp.Info("MQTT Client connected: %t", input.client.IsConnected())
	return nil
}

func (input *mqttInput) subscribeOnConnect(client MQTT.Client) {
	subscriptions := input.parseTopics(input.config.Topics, input.config.QoS)
	//bt.beatConfig.TopicsSubscribe
	logp.Info("Current status: %v", input.client.IsConnected())

	// Mqtt client - Subscribe to every topic in the config file, and bind with message handler
	if token := input.client.SubscribeMultiple(subscriptions, input.onMessage); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	logp.Info("Subscribed to configured topics")
}

// Mqtt message handler
func (input *mqttInput) onMessage(client MQTT.Client, msg MQTT.Message) {
	logp.Debug("MQTT Module", "MQTT message received: %s", string(msg.Payload()))
	var beatEvent beat.Event
	event := make(common.MapStr)

	// default case
	var message = make(common.MapStr)
	event["message"] = string(msg.Payload())
	if input.config.DecodePaylod == true {
		message["fields"] = DecodePayload(msg.Payload())
	}

	if strings.HasPrefix(msg.Topic(), "$") {
		event["isSystemTopic"] = true
	} else {
		event["isSystemTopic"] = false
	}
	event["topic"] = msg.Topic()
	message["ID"] = msg.MessageID()
	message["retained"] = msg.Retained()
	event["mqtt"] = message

	// Finally sending the message to elasticsearch
	beatEvent.Fields = event

	input.outlet.OnEvent(beatEvent)

	logp.Debug("MQTT", "Event sent: %t")
}

// DefaultConnectionLostHandler does nothing
func (input *mqttInput) reConnectHandler(client MQTT.Client, reason error) {
	logp.Warn("[MQTT] Connection lost: %s", reason.Error())
}

// DecodePayload will try to decode the payload. If every check fails, it will
// return the payload as a string
func DecodePayload(payload []byte) common.MapStr {
	event := make(common.MapStr)

	// A msgpack payload must be a json-like object
	err := msgpack.Unmarshal(payload, &event)
	if err == nil {
		logp.Debug("mqttbeat", "Payload decoded - msgpack")
		return event
	}

	err = json.Unmarshal(payload, &event)
	if err == nil {
		logp.Debug("mqttbeat", "Payload decoded - json")
		return event
	}

	logp.Debug("mqttbeat", "decoded - text")
	return event
}

// ParseTopics will parse the config file and return a map with topic:QoS
func (input *mqttInput) parseTopics(topics []string, qos int) map[string]byte {
	subscriptions := make(map[string]byte)
	for _, value := range topics {
		// Finally, filling the subscriptions map
		subscriptions[value] = byte(qos)
		logp.Info("Subscribe to %v with QoS %v", value, qos)
	}
	return subscriptions
}
