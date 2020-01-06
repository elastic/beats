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
	"fmt"
	"time"
)

type mqttInputConfig struct {
	Host           string        `config:"host"`
	Topics         []string      `config:"topics"`
	Username       string        `config:"user"`
	Password       string        `config:"password"`
	QoS            int           `config:"QoS"`
	DecodePayload  bool          `config:"decode_payload"`
	SSL            bool          `config:"ssl"`
	CA             string        `config:"CA"`
	ClientCert     string        `config:"clientCert"`
	ClientKey      string        `config:"clientKey"`
	ClientID       string        `config:"clientID"`
	WaitClose      time.Duration `config:"wait_close" validate:"min=0"`
	ConnectBackoff time.Duration `config:"connect_backoff" validate:"min=0"`
}

// The default config for the mqtt input
func defaultConfig() mqttInputConfig {
	return mqttInputConfig{
		Host:           "localhost",
		Topics:         []string{"#"},
		ClientID:       "Filebeat",
		Username:       "",
		Password:       "",
		DecodePayload:  true,
		QoS:            0,
		SSL:            false,
		CA:             "",
		ClientCert:     "",
		ClientKey:      "",
		WaitClose:      5 * time.Second,
		ConnectBackoff: 30 * time.Second,
	}
}

// Validate validates the config.
func (c *mqttInputConfig) Validate() error {
	if c.Host == "" {
		return errors.New("no host configured")
	}

	if c.Username != "" && c.Password == "" {
		return fmt.Errorf("password must be set when username is configured")
	}

	if len(c.ClientID) > 23 || len(c.ClientID) < 1 {
		return fmt.Errorf("client id must be between 1 and 23 characters long")
	}
	return nil
}
