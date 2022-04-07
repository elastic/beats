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

	"github.com/elastic/beats/v8/libbeat/common/transport/tlscommon"
)

type mqttInputConfig struct {
	Hosts  []string `config:"hosts" validate:"required,min=1"`
	Topics []string `config:"topics" validate:"required,min=1"`
	QoS    int      `config:"qos" validate:"min=0,max=2"`

	ClientID string `config:"client_id" validate:"nonzero"`
	Username string `config:"username"`
	Password string `config:"password"`

	TLS *tlscommon.Config `config:"ssl"`
}

// The default config for the mqtt input.
func defaultConfig() mqttInputConfig {
	return mqttInputConfig{
		ClientID: "filebeat",
		Topics:   []string{"#"},
	}
}

// Validate validates the config.
func (mic *mqttInputConfig) Validate() error {
	if len(mic.ClientID) < 1 || len(mic.ClientID) > 23 {
		return errors.New("ClientID must be between 1 and 23 characters long")
	}
	return nil
}
