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

package logv2

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/filebeat/channel"
	v1 "github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/input/filestream"
	loginput "github.com/elastic/beats/v7/filebeat/input/log"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/features"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

const pluginName = "log"

func init() {
	// Register an input V1, to replace the Log input one.
	if err := v1.Register(pluginName, newV1Input); err != nil {
		panic(err)
	}
}

// runAsFilestream validates cfg as a Log input configuration, if it is
// valid, then checks whether the configuration should be run as
// Filestream input. On any error the boolean value must be ignored and
// no input started. runAsFilestream also sets the input type accordingly.
func runAsFilestream(logger *logp.Logger, cfg *config.C) (bool, error) {
	// First of all, ensure the Log input configuration is valid.
	// This ensures we return configuration errors compatible
	// with the log input.
	if err := loginput.IsConfigValid(cfg); err != nil {
		return false, err
	}

	// Global feature flag that forces all Log input instances
	// to run as Filestream, even when not running under
	// Elastic Agent
	if features.LogInputRunFilestream() {
		return true, nil
	}

	// We don't allow to run the Log input as Filestream if Filebeat
	// is not running under Elastic Agent.
	if !management.UnderAgent() {
		return false, nil
	}

	if ok := cfg.HasField("run_as_filestream"); ok {
		runAsFilestream, err := cfg.Bool("run_as_filestream", -1)
		if err != nil {
			return false, fmt.Errorf("cannot parse 'run_as_filestream': %w", err)
		}

		if runAsFilestream {
			// ID is required to run as Filestream input
			if !cfg.HasField("id") {
				logger.Warnf(
					"'id' is required to run 'log' input as 'filestream'. Config: %s",
					config.DebugString(cfg, false),
				)
				return false, errors.New("'id' is required to run 'log' input as 'filestream'")
			}

			// This should never fail because the Log input configuration
			// already reads 'type' and validates it is a string. Overriding
			// the field should never fail.
			if err := cfg.SetString("type", -1, "filestream"); err != nil {
				return false, fmt.Errorf("cannot set 'type': %w", err)
			}

			return true, nil
		}
	}

	return false, nil
}

// newV1Input instantiates the Log input. If Log input is supposed to run as
// Filestream, then v2.ErrUnknownInput is returned so the Filestream input
// can be instantiated by the V2.Plugin returned by [PluginV2]. Otherwise
// the Log input is instantiated.
func newV1Input(
	cfg *config.C,
	outlet channel.Connector,
	context v1.Context,
	logger *logp.Logger,
) (v1.Input, error) {
	asFilestream, err := runAsFilestream(logger, cfg)
	if err != nil {
		return nil, err
	}

	if asFilestream {
		return nil, v2.ErrUnknownInput
	}

	// Add the input ID to the logger, if it exists
	if id, err := cfg.String("id", -1); err == nil {
		logger = logger.With("id", id)
	}

	inp, err := loginput.NewInput(cfg, outlet, context, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create log input: %w", err)
	}

	logger.Info("Log input (deprecated) running as Log input (deprecated)")
	return inp, err
}

// PluginV2 returns a v2.Plugin with a manager that can convert
// the Log input configuration to Filestream and run the Filestream
// input instead of the Log input.
func PluginV2(logger *logp.Logger, store statestore.States) v2.Plugin {
	// The InputManager for Filestream input is from an internal package, so we
	// cannot instantiate it directly here. To circumvent that, we instantiate
	// the whole Filestream Plugin and get its manager.
	filestreamPlugin := filestream.Plugin(logger, store)

	m := manager{
		next:   filestreamPlugin.Manager,
		logger: logger,
	}

	p := v2.Plugin{
		Name:      pluginName,
		Stability: feature.Stable,
		Info:      "log input running filestream",
		Doc:       "Log input running Filestream input",
		Manager:   m,
	}
	return p
}

type manager struct {
	next   v2.InputManager
	logger *logp.Logger
}

func (m manager) Init(grp unison.Group) error {
	return m.next.Init(grp)
}

// Create first checks whether the config is supposed to run as Filestream
// and creates the Filestream input if needed.
// If the configuration is not supposed to run as Filestream,
// v2.ErrUnknownInput is returned.
func (m manager) Create(cfg *config.C) (v2.Input, error) {
	asFilestream, err := runAsFilestream(m.logger, cfg)
	if err != nil {
		return nil, err
	}

	if !asFilestream {
		return nil, v2.ErrUnknownInput
	}

	newCfg, err := convertConfig(m.logger, cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot translate log config to filestream: %w", err)
	}

	// We know 'id' exists in the config and can be retrieved because
	// 'runAsFilestream' has already validated it, hence it is safe to
	// ignore the error.
	id, _ := cfg.String("id", -1)
	m.logger.Infow("Log input (deprecated) running as Filestream input", "id", id)
	return m.next.Create(newCfg)
}
