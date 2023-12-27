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

package fileset

import (
	"fmt"
	"path/filepath"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/paths"
)

// ModuleConfig contains the configuration file options for a module
type ModuleConfig struct {
	Module  string `config:"module"     validate:"required"`
	Enabled *bool  `config:"enabled"`

	// Filesets is inlined by code, see mcfgFromConfig
	Filesets map[string]*FilesetConfig
}

// FilesetConfig contains the configuration file options for a fileset
type FilesetConfig struct {
	Enabled *bool                  `config:"enabled"`
	Var     map[string]interface{} `config:"var"`
	Input   map[string]interface{} `config:"input"`
}

// NewFilesetConfig creates a new FilesetConfig from a conf.C.
func NewFilesetConfig(cfg *conf.C) (*FilesetConfig, error) {
	if err := cfgwarn.CheckRemoved6xSetting(cfg, "prospector"); err != nil {
		return nil, err
	}

	var fcfg FilesetConfig
	err := cfg.Unpack(&fcfg)
	if err != nil {
		return nil, fmt.Errorf("error unpacking configuration")
	}

	return &fcfg, nil
}

// mergePathDefaults returns a copy of c containing the path variables that must
// be available for variable expansion in module configuration (e.g. it enables
// the use of ${path.config} in module config).
func mergePathDefaults(c *conf.C) (*conf.C, error) {
	defaults := conf.MustNewConfigFrom(map[string]interface{}{
		"path": map[string]interface{}{
			"home":   paths.Paths.Home,
			"config": "${path.home}",
			"data":   filepath.Join("${path.home}", "data"),
			"logs":   filepath.Join("${path.home}", "logs"),
		},
	})
	if err := defaults.Merge(c); err != nil {
		return nil, err
	}
	return defaults, nil
}
