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

import "github.com/menderesk/beats/v7/libbeat/common"

type config struct {
	Key           string         `config:"key"`
	DefaultConfig *common.Config `config:"default_config"`
}

func defaultConfig() config {
	defaultCfgRaw := map[string]interface{}{
		"type": "container",
		"paths": []string{
			// To be able to use this builder with CRI-O replace paths with:
			// /var/log/pods/${data.kubernetes.pod.uid}/${data.kubernetes.container.name}/*.log
			"/var/lib/docker/containers/${data.container.id}/*-json.log",
		},
	}
	defaultCfg, _ := common.NewConfigFrom(defaultCfgRaw)
	return config{
		Key:           "logs",
		DefaultConfig: defaultCfg,
	}
}

func (c *config) Unpack(from *common.Config) error {
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
				return nil
			}
		} else {
			// full config provided, discard default. It must be a clone of the
			// given config otherwise it could be updated across multiple inputs.
			c.DefaultConfig = common.MustNewConfigFrom(config)
		}
	}

	c.Key = tmpConfig.Key
	return nil
}
