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

// NewAgentConfig allows the packetbeat configuration to understand
// agent semantics
func NewAgentConfig(cfg *common.Config) (Config, error) {
	logp.Debug("agent", "Normalizing agent configuration")
	var configMap []map[string]interface{}
	config := Config{
		Interfaces: InterfacesConfig{
			// TODO: make this configurable rather than just using the default device
			Device: defaultDevice(),
		},
	}
	if err := cfg.Unpack(&configMap); err != nil {
		return config, err
	}

	logp.Debug("agent", fmt.Sprintf("Found %d inputs", len(configMap)))
	for _, input := range configMap {
		if rawInputType, ok := input["type"]; ok {
			inputType, ok := rawInputType.(string)
			if !ok {
				return config, fmt.Errorf("invalid input type of: '%T'", rawInputType)
			}
			logp.Debug("agent", fmt.Sprintf("Found agent configuration for %v", inputType))
			if strings.HasPrefix(inputType, "network/") {
				cfg, err := common.NewConfigFrom(input)
				if err != nil {
					return config, err
				}
				protocol := strings.TrimPrefix(inputType, "network/")
				switch protocol {
				case "flows":
					if err := cfg.Unpack(&config.Flows); err != nil {
						return config, err
					}
				default:
					if err = cfg.SetString("type", -1, protocol); err != nil {
						return config, err
					}
					config.ProtocolsList = append(config.ProtocolsList, cfg)
				}
			}
		}
	}
	return config, nil
}
