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
	libmqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/elastic/beats/v8/libbeat/common/transport/tlscommon"
)

func createClientOptions(config mqttInputConfig, onConnectHandler func(client libmqtt.Client)) (*libmqtt.ClientOptions, error) {
	clientOptions := libmqtt.NewClientOptions().
		SetClientID(config.ClientID).
		SetUsername(config.Username).
		SetPassword(config.Password).
		SetConnectRetry(true).
		SetOnConnectHandler(onConnectHandler)

	for _, host := range config.Hosts {
		clientOptions.AddBroker(host)
	}

	if config.TLS != nil {
		tlsConfig, err := tlscommon.LoadTLSConfig(config.TLS)
		if err != nil {
			return nil, err
		}
		clientOptions.SetTLSConfig(tlsConfig.BuildModuleClientConfig(""))
	}
	return clientOptions, nil
}

func createClientSubscriptions(config mqttInputConfig) map[string]byte {
	subscriptions := map[string]byte{}
	for _, topic := range config.Topics {
		subscriptions[topic] = byte(config.QoS)
	}
	return subscriptions
}
