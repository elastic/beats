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

package emptydir

import (
	"github.com/elastic/beats/libbeat/common"
)

type inputConfig struct {
	RootDir       string         `config:"root_dir"`
	Key           string         `config:"key"`
	DefaultConfig *common.Config `config:"default_config"`
}

func defaultConfig() inputConfig {
	return inputConfig{
		RootDir:       "/var/lib/kubelet/pods/",
		Key:           "logs",
		DefaultConfig: getBaseConfig(),
	}
}

func getBaseConfig() *common.Config {
	config := common.MapStr{
		"type":  "log",
		"paths": "${data.paths}",
	}

	cfg, _ := common.NewConfigFrom(&config)
	return cfg
}

// Unpack is needed here as go-ucfg fails to unpack the Config object by default
func (c *inputConfig) Unpack(from *common.Config) error {
	tmpConfig := struct {
		RootDir string `config:"root_dir"`
		Key     string `config:"key"`
	}{
		Key:     c.Key,
		RootDir: c.RootDir,
	}
	if err := from.Unpack(&tmpConfig); err != nil {
		return err
	}

	if config, err := from.Child("default_config", -1); err == nil {
		c.DefaultConfig = config
	}

	c.Key = tmpConfig.Key
	c.RootDir = tmpConfig.RootDir
	return nil
}
