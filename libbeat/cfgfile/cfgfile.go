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

package cfgfile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Command line flags.
var (
	// The default config cannot include the beat name as it is not initialized
	// when this variable is created. See ChangeDefaultCfgfileFlag which should
	// be called prior to flags.Parse().
	configfiles = config.StringArrFlag(nil, "c", "beat.yml", "Configuration file, relative to path.config")
	overwrites  = config.SettingFlag(nil, "E", "Configuration overwrite")

	// Additional default settings, that must be available for variable expansion
	defaults = config.MustNewConfigFrom(map[string]interface{}{
		"path": map[string]interface{}{
			"home":   ".", // to be initialized by beat
			"config": "${path.home}",
			"data":   fmt.Sprint("${path.home}", string(os.PathSeparator), "data"),
			"logs":   fmt.Sprint("${path.home}", string(os.PathSeparator), "logs"),
		},
	})

	// home-path CLI flag (initialized in init)
	homePath   *string
	configPath *string
)

func init() {
	// add '-path.x' options overwriting paths in 'overwrites' config
	makePathFlag := func(name, usage string) *string {
		return config.ConfigOverwriteFlag(nil, overwrites, name, name, "", usage)
	}

	homePath = makePathFlag("path.home", "Home path")
	configPath = makePathFlag("path.config", "Configuration path")
	makePathFlag("path.data", "Data path")
	makePathFlag("path.logs", "Logs path")
}

// OverrideChecker checks if a config should be overwritten.
type OverrideChecker func(*config.C) bool

// ConditionalOverride stores a config which needs to overwrite the existing config based on the
// result of the Check.
type ConditionalOverride struct {
	Check  OverrideChecker
	Config *config.C
}

// ChangeDefaultCfgfileFlag replaces the value and default value for the `-c`
// flag so that it reflects the beat name.
func ChangeDefaultCfgfileFlag(beatName string) error {
	configfiles.SetDefault(beatName + ".yml")
	return nil
}

// GetDefaultCfgfile gets the full path of the default config file. Understood
// as the first value for the `-c` flag. By default this will be `<beatname>.yml`
func GetDefaultCfgfile() string {
	if len(configfiles.List()) == 0 {
		return ""
	}

	cfg := configfiles.List()[0]
	cfgpath := GetPathConfig()

	if !filepath.IsAbs(cfg) {
		return filepath.Join(cfgpath, cfg)
	}
	return cfg
}

// HandleFlags adapts default config settings based on command line flags.
func HandleFlags() error {
	// default for the home path is the binary location
	home, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		if *homePath == "" {
			return fmt.Errorf("The absolute path to %s could not be obtained. %v",
				os.Args[0], err)
		}
		home = *homePath
	}

	defaults.SetString("path.home", -1, home)

	if len(overwrites.GetFields()) > 0 {
		common.PrintConfigDebugf(overwrites, "CLI setting overwrites (-E flag):")
	}

	return nil
}

// Deprecated: Please use Load().
//
// Read reads the configuration from a YAML file into the given interface
// structure. If path is empty this method reads from the configuration
// file specified by the '-c' command line flag.
func Read(out interface{}, path string) error {
	config, err := Load(path, nil)
	if err != nil {
		return err
	}

	return config.Unpack(out)
}

// Load reads the configuration from a YAML file structure. If path is empty
// this method reads from the configuration file specified by the '-c' command
// line flag.
func Load(path string, beatOverrides []ConditionalOverride) (*config.C, error) {
	var c *config.C
	var err error

	cfgpath := GetPathConfig()

	if path == "" {
		list := []string{}
		for _, cfg := range configfiles.List() {
			if !filepath.IsAbs(cfg) {
				list = append(list, filepath.Join(cfgpath, cfg))
			} else {
				list = append(list, cfg)
			}
		}
		c, err = common.LoadFiles(list...)
	} else {
		if !filepath.IsAbs(path) {
			path = filepath.Join(cfgpath, path)
		}
		c, err = common.LoadFile(path)
	}
	if err != nil {
		return nil, err
	}

	if beatOverrides != nil {
		merged := defaults
		for _, o := range beatOverrides {
			if o.Check(c) {
				merged, err = config.MergeConfigs(merged, o.Config)
				if err != nil {
					return nil, err
				}
			}
		}
		c, err = config.MergeConfigs(
			merged,
			c,
			overwrites,
		)
		if err != nil {
			return nil, err
		}
	} else {
		c, err = config.MergeConfigs(
			defaults,
			c,
			overwrites,
		)
	}

	common.PrintConfigDebugf(c, "Complete configuration loaded:")
	return c, nil
}

// LoadList loads a list of configs data from the given file.
func LoadList(file string) ([]*config.C, error) {
	logp.Debug("cfgfile", "Load config from file: %s", file)
	rawConfig, err := common.LoadFile(file)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %s", err)
	}

	var c []*config.C
	err = rawConfig.Unpack(&c)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration from file %s: %s", file, err)
	}

	return c, nil
}

func SetConfigPath(path string) {
	*configPath = path
}

// GetPathConfig returns ${path.config}. If ${path.config} is not set, ${path.home} is returned.
func GetPathConfig() string {
	if *configPath != "" {
		return *configPath
	} else if *homePath != "" {
		return *homePath
	}
	// TODO: Do we need this or should we always return *homePath?
	return ""
}
