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
	"os"
	"sort"
	"time"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// Defaults for config variables which are not set
const (
	DefaultType = "log"
)

type Config struct {
	Inputs             []*conf.C            `config:"inputs"`
	Registry           Registry             `config:"registry"`
	ConfigDir          string               `config:"config_dir"`
	ShutdownTimeout    time.Duration        `config:"shutdown_timeout"`
	Modules            []*conf.C            `config:"modules"`
	ConfigInput        *conf.C              `config:"config.inputs"`
	ConfigModules      *conf.C              `config:"config.modules"`
	Autodiscover       *autodiscover.Config `config:"autodiscover"`
	OverwritePipelines bool                 `config:"overwrite_pipelines"`
}

type Registry struct {
	Path          string        `config:"path"`
	Permissions   os.FileMode   `config:"file_permissions"`
	FlushTimeout  time.Duration `config:"flush"`
	CleanInterval time.Duration `config:"cleanup_interval"`
	MigrateFile   string        `config:"migrate_file"`
}

var DefaultConfig = Config{
	Registry: Registry{
		Path:          "registry",
		Permissions:   0o600,
		MigrateFile:   "",
		CleanInterval: 5 * time.Minute,
		FlushTimeout:  time.Second,
	},
	ShutdownTimeout:    0,
	OverwritePipelines: false,
}

// ListEnabledInputs returns a list of enabled inputs sorted by alphabetical order.
func (config *Config) ListEnabledInputs() []string {
	t := struct {
		Type string `config:"type"`
	}{}
	var inputs []string
	for _, input := range config.Inputs {
		if input.Enabled() {
			_ = input.Unpack(&t)
			inputs = append(inputs, t.Type)
		}
	}
	sort.Strings(inputs)
	return inputs
}

// IsInputEnabled returns true if the plugin name is enabled.
func (config *Config) IsInputEnabled(name string) bool {
	enabledInputs := config.ListEnabledInputs()
	for _, input := range enabledInputs {
		if name == input {
			return true
		}
	}
	return false
}
