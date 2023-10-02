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

package filestream

import (
	"fmt"
	"strings"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const containerPluginName = "container-v2"

func defaultContainerConfig() containerConfig {
	return containerConfig{
		Stream: "all",
		Format: "auto",
	}
}

type containerConfig struct {
	// Stream can be all, stdout or stderr
	Stream string `config:"stream"`

	// Format can be auto, cri, json-file
	Format string `config:"format"`
}

// Validate validates the config.
func (c *containerConfig) Validate() error {
	if !stringInSlice(c.Stream, []string{"all", "stdout", "stderr"}) {
		return fmt.Errorf("invalid value for stream: %s, supported values are: all, stdout, stderr", c.Stream)
	}

	if !stringInSlice(strings.ToLower(c.Format), []string{"auto", "docker", "cri"}) {
		return fmt.Errorf("invalid value for format: %s, supported values are: auto, docker, cri", c.Format)
	}

	return nil
}

// Plugin creates a new container V2 input plugin for creating a stateful input.
func ContainerPlugin(log *logp.Logger, store loginp.StateStore) input.Plugin {
	return input.Plugin{
		Name:       containerPluginName,
		Stability:  feature.Beta,
		Deprecated: false,
		Info:       "filestream-based container input",
		Doc:        "The container input collects logs from a running container using filestream",
		Manager: &loginp.InputManager{
			Logger:     log,
			StateStore: store,
			Type:       containerPluginName,
			Configure:  configureContainer,
		},
	}
}

type container struct {
	loginp.Harvester
}

func (container) Name() string { return containerPluginName }

func configureContainer(cfg *conf.C) (loginp.Prospector, loginp.Harvester, error) {
	containerConfig := defaultContainerConfig()
	if err := cfg.Unpack(&containerConfig); err != nil {
		return nil, nil, fmt.Errorf("failed to read container input config: %w", err)
	}

	err := cfg.Merge(mapstr.M{
		"parsers": []mapstr.M{
			{
				"container.stream": containerConfig.Stream,
				"container.format": containerConfig.Format,
			},
		},
		// Set symlinks to true as CRI-O paths could point to symlinks instead of the actual path.
		"prospector.scanner.symlinks": true,
		// Most of the time container logs are ingested from file systems without stable inode values
		"prospector.scanner.fingerprint.enabled": true,
		"file_identity.fingerprint":              nil,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update container input config: %w", err)
	}

	prospector, harvester, err := configure(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create filestream for container input: %w", err)
	}

	return prospector, container{harvester}, nil
}

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}
