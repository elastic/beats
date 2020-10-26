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

package config

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

var osDefaultDevices = map[string]string{
	"darwin": "en0",
	"linux":  "any",
}

func defaultDevice() string {
	if device, found := osDefaultDevices[runtime.GOOS]; found {
		return device
	}
	return "0"
}

// Normalize allows the packetbeat configuration to understand
// agent semantics
func (c Config) Normalize() (Config, error) {
	logp.Debug("agent", "Normalizing agent configuration")
	if len(c.Inputs) > 0 {
		// override everything, we're managed by agent
		c.Flows = nil
		c.Protocols = nil
		c.ProtocolsList = []*common.Config{}
		// TODO: make this configurable rather than just using the default device in
		// managed mode
		c.Interfaces.Device = defaultDevice()
	}

	for _, input := range c.Inputs {
		if rawInputType, ok := input["type"]; ok {
			inputType, ok := rawInputType.(string)
			if ok && strings.HasPrefix(inputType, "network/") {
				config, err := common.NewConfigFrom(input)
				if err != nil {
					return c, err
				}
				protocol := strings.TrimPrefix(inputType, "network/")
				logp.Debug("agent", fmt.Sprintf("Found agent configuration for %v", protocol))
				switch protocol {
				case "flows":
					if err := config.Unpack(&c.Flows); err != nil {
						return c, err
					}
				default:
					if err = config.SetString("type", -1, protocol); err != nil {
						return c, err
					}
					c.ProtocolsList = append(c.ProtocolsList, config)
				}
			}
		}
	}
	return c, nil
}
