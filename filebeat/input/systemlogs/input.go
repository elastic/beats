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

package systemlogs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/filebeat/channel"
	v1 "github.com/elastic/beats/v7/filebeat/input"
	loginput "github.com/elastic/beats/v7/filebeat/input/log"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/feature"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const pluginName = "system-logs"

func init() {
	// Register an input V1, that's used by the log input
	if err := v1.Register(pluginName, newV1Input); err != nil {
		panic(err)
	}
}

type config struct {
	UseJournald bool    `config:"use_journald"`
	UseFiles    bool    `config:"use_files"`
	Files       *conf.C `config:"files" yaml:"files"`
	Journald    *conf.C `config:"journald" yaml:"journald"`
}

func (c *config) Validate() error {
	if c.UseFiles && c.UseJournald {
		return errors.New("'use_journald' and 'use_files' cannot both be true")
	}

	if c.Files == nil && c.Journald == nil {
		return errors.New("one of 'journald' or 'files' must be set")
	}

	return nil
}

// newV1Input checks whether the log input must be created and
// delegates to loginput.NewInput if needed.
func newV1Input(
	cfg *conf.C,
	outlet channel.Connector,
	context v1.Context,
) (v1.Input, error) {
	journald, err := useJournald(cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot decide between journald and files: %w", err)
	}

	if journald {
		return nil, v2.ErrUnknownInput
	}

	// Convert the configuration and create a log input
	logCfg, err := toFilesConfig(cfg)
	if err != nil {
		return nil, err
	}

	return loginput.NewInput(logCfg, outlet, context)
}

// PluginV2 creates a v2.Plugin that will instantiate a journald
// input if needed.
func PluginV2(logger *logp.Logger, store cursor.StateStore) v2.Plugin {
	logger = logger.Named(pluginName)

	return v2.Plugin{
		Name:       pluginName,
		Stability:  feature.Experimental,
		Deprecated: false,
		Info:       "system-logs input",
		Doc:        "The system-logs input collects system logs on Linux by reading them from journald or traditional log files",
		Manager: &cursor.InputManager{
			Logger:     logger,
			StateStore: store,
			Type:       pluginName,
			Configure:  configure,
		},
	}
}

// useJournald returns true if jounrald should be used.
// If there is an error, false is always retruned.
//
// The decision logic is:
//   - If UseJournald is set, return true
//   - If UseFiles is set, return false
//   - If the globs defined in `files.paths` match any existing file,
//     return false
//   - Otherwise return true
func useJournald(c *conf.C) (bool, error) {
	logger := logp.L().Named("input.system-logs")

	cfg := config{}
	if err := c.Unpack(&cfg); err != nil {
		return false, fmt.Errorf("cannot unpack 'system-logs' config: %w", err)
	}

	if cfg.UseJournald {
		logger.Info("using journald input because 'use_journald' is set")
		return true, nil
	}

	if cfg.UseFiles {
		logger.Info("using log input because 'use_files' is set")
		return false, nil
	}

	globs := struct {
		Paths []string `config:"files.paths"`
	}{}

	if err := c.Unpack(&globs); err != nil {
		return false, fmt.Errorf("cannot parse paths from config: %w", err)
	}

	for _, g := range globs.Paths {
		paths, err := filepath.Glob(g)
		if err != nil {
			return false, fmt.Errorf("cannot resolve glob: %w", err)
		}

		for _, p := range paths {
			stat, err := os.Stat(p)
			if err != nil {
				return false, fmt.Errorf("cannot stat '%s': %w", p, err)
			}

			// Ignore directories
			if stat.IsDir() {
				continue
			}

			// We found one file, return early
			logger.Infof(
				"using log input because file(s) was(were) found when testing glob '%s'",
				g)
			return false, nil
		}
	}

	// if no system log files are found, then use jounrald
	logger.Info("no files were found, using journald input")

	return true, nil
}

func toFilesConfig(cfg *conf.C) (*conf.C, error) {
	newCfg, err := cfg.Child("files", -1)
	if err != nil {
		return nil, fmt.Errorf("cannot extract 'journald' block: %w", err)
	}

	if _, err := cfg.Remove("journald", -1); err != nil {
		return nil, err
	}

	if _, err := cfg.Remove("type", -1); err != nil {
		return nil, err
	}

	if _, err := cfg.Remove("files", -1); err != nil {
		return nil, err
	}

	if _, err := cfg.Remove("use_journald", -1); err != nil {
		return nil, err
	}

	if _, err := cfg.Remove("use_files", -1); err != nil {
		return nil, err
	}

	if err := newCfg.Merge(cfg); err != nil {
		return nil, err
	}

	if err := newCfg.SetString("type", -1, "log"); err != nil {
		return nil, fmt.Errorf("cannot set 'type': %w", err)
	}

	if err := newCfg.SetBool("allow_deprecated_use", -1, true); err != nil {
		return nil, fmt.Errorf("cannot set 'allow_deprecated_use': %w", err)
	}

	if err := cfg.SetString("type", -1, pluginName); err != nil {
		return nil, fmt.Errorf("cannot set type back to '%s': %w", pluginName, err)
	}

	return newCfg, nil
}
