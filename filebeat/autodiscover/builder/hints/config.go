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

package hints

import (
	conf "github.com/elastic/elastic-agent-libs/config"
)

type config struct {
	Key           string  `config:"key"`
	DefaultConfig *conf.C `config:"default_config"`
}

func defaultConfig() config {
	defaultCfgRaw := map[string]interface{}{
		"type": "filestream",
		"id":   "kubernetes-container-logs-${data.kubernetes.container.id}",
		"prospector": map[string]interface{}{
			"scanner": map[string]interface{}{
				"fingerprint.enabled": true,
				"symlinks":            true,
			},
		},
		"file_identity.fingerprint": nil,
		"parsers": []interface{}{
			map[string]interface{}{
				"container": map[string]interface{}{
					"stream": "all",
					"format": "auto",
				},
			},
		},
		"paths": []string{
			"/var/log/containers/*-${data.kubernetes.container.id}.log",
		},
	}
	defaultCfg, _ := conf.NewConfigFrom(defaultCfgRaw)
	return config{
		Key:           "logs",
		DefaultConfig: defaultCfg,
	}
}

func (c *config) Unpack(from *conf.C) error {
	tmpConfig := struct {
		Key string `config:"key"`
	}{
		Key: c.Key,
	}
	if err := from.Unpack(&tmpConfig); err != nil {
		return err
	}

	if config, err := from.Child("default_config", -1); err == nil {
		fields := config.GetFields()
		if len(fields) == 1 && fields[0] == "enabled" {
			// only enabling/disabling default config:
			if err := c.DefaultConfig.Merge(config); err != nil {
				return err
			}
		} else {
			// full config provided, discard default. It must be a clone of the
			// given config otherwise it could be updated across multiple inputs.
			c.DefaultConfig = conf.MustNewConfigFrom(config)
		}
	}

	c.Key = tmpConfig.Key
	return nil
}
