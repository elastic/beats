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

	"github.com/elastic/beats/v7/filebeat/channel"
	v1 "github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/input/journald"
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

// newV1Input creates a new log input
func newV1Input(
	cfg *conf.C,
	outlet channel.Connector,
	context v1.Context,
) (v1.Input, error) {

	useJournald, cfg, err := decide(cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot decide between journald and files: %w", err)
	}

	if !useJournald {
		inp, err := loginput.NewInput(cfg, outlet, context)
		if err != nil {
			return nil, fmt.Errorf("cannot create log input: %w", err)
		}
		return inp, err
	}

	return nil, v2.ErrUnknownInput
}

// PluginV2 creates a v2 plugin that will instantiate a journald
// input if needed.
func PluginV2(logger *logp.Logger, store cursor.StateStore) v2.Plugin {
	logger = logger.Named(pluginName)

	return v2.Plugin{
		Name:       pluginName,
		Stability:  feature.Stable,
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

// configure checks whether the journald input must be created and
// delegates to journald.Configure if needed.
func configure(cfg *conf.C) ([]cursor.Source, cursor.Input, error) {
	useJournald, cfg, err := decide(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot decide between journald and files: %w", err)
	}

	if useJournald {
		return journald.Configure(cfg)
	}

	return nil, nil, errors.New("cannot initialise system-logs with journald input")
}

// decide returns:
//   - use Jounrald (input V2)
//   - the new config
//   - error, if any
func decide(c *conf.C) (bool, *conf.C, error) {
	cfg := config{}
	if err := c.Unpack(&cfg); err != nil {
		return false, nil, err
	}

	if cfg.UseJournald {
		cfg, err := toJournaldConfig(c)
		return true, cfg, err
	}

	if cfg.UseFiles {
		cfg, err := toFilesConfig(c)
		return false, cfg, err
	}

	// TODO: implement checking the files

	return false, nil, errors.New("[WIP] either set use_journald or use_files")
}

func toJournaldConfig(cfg *conf.C) (*conf.C, error) {
	newCfg, err := cfg.Child("journald", -1)
	if err != nil {
		return nil, fmt.Errorf("cannot extract 'journald' block: %w", err)
	}

	if err := newCfg.SetString("type", -1, "journald"); err != nil {
		return nil, fmt.Errorf("cannot set 'type': %w", err)
	}

	return newCfg, nil
}

func toFilesConfig(cfg *conf.C) (*conf.C, error) {
	newCfg, err := cfg.Child("files", -1)
	if err != nil {
		return nil, fmt.Errorf("cannot extract 'journald' block: %w", err)
	}

	if err := newCfg.SetString("type", -1, "log"); err != nil {
		return nil, fmt.Errorf("cannot set 'type': %w", err)
	}

	return newCfg, nil
}
