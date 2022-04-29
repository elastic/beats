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
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/paths"
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
	},
	ShutdownTimeout:    0,
	OverwritePipelines: false,
}

// getConfigFiles returns list of config files.
// In case path is a file, it will be directly returned.
// In case it is a directory, it will fetch all .yml files inside this directory
func getConfigFiles(path string) (configFiles []string, err error) {
	// Check if path is valid file or dir
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// Create empty slice for config file list
	configFiles = make([]string, 0)

	if stat.IsDir() {
		files, err := filepath.Glob(path + "/*.yml")
		if err != nil {
			return nil, err
		}

		configFiles = append(configFiles, files...)

	} else {
		// Only 1 config file
		configFiles = append(configFiles, path)
	}

	return configFiles, nil
}

// mergeConfigFiles reads in all config files given by list configFiles and merges them into config
func mergeConfigFiles(configFiles []string, config *Config) error {
	for _, file := range configFiles {
		logp.Info("Additional configs loaded from: %s", file)

		tmpConfig := struct {
			Filebeat Config
		}{}
		err := cfgfile.Read(&tmpConfig, file)
		if err != nil {
			return fmt.Errorf("Failed to read %s: %s", file, err)
		}

		config.Inputs = append(config.Inputs, tmpConfig.Filebeat.Inputs...)
	}

	return nil
}

// Fetches and merges all config files given by configDir. All are put into one config object
func (config *Config) FetchConfigs() error {
	configDir := config.ConfigDir

	// If option not set, do nothing
	if configDir == "" {
		return nil
	}

	cfgwarn.Deprecate("7.0.0", "config_dir is deprecated. Use `filebeat.config.inputs` instead.")

	// If configDir is relative, consider it relative to the config path
	configDir = paths.Resolve(paths.Config, configDir)

	// Check if optional configDir is set to fetch additional config files
	logp.Info("Additional config files are fetched from: %s", configDir)

	configFiles, err := getConfigFiles(configDir)
	if err != nil {
		log.Fatal("Could not use config_dir of: ", configDir, err)
		return err
	}

	err = mergeConfigFiles(configFiles, config)
	if err != nil {
		log.Fatal("Error merging config files: ", err)
		return err
	}

	return nil
}

// ListEnabledInputs returns a list of enabled inputs sorted by alphabetical order.
func (config *Config) ListEnabledInputs() []string {
	t := struct {
		Type string `config:"type"`
	}{}
	var inputs []string
	for _, input := range config.Inputs {
		if input.Enabled() {
			input.Unpack(&t)
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
