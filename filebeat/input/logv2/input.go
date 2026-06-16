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
	loginput "github.com/elastic/beats/v7/filebeat/input/log"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/features"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

const logPluginName = "log"
const containerPluginName = "container"

func init() {
	// Register an input V1, to replace the Log input one.
	if err := v1.Register(logPluginName, NewV1Input); err != nil {
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

			return true, nil
		}
	}

	return false, nil
}

// NewV1Input instantiates the Log input. If Log input is supposed to run as
// Filestream, then v2.ErrUnknownInput is returned so the Filestream input
// can be instantiated by the V2.Plugin returned by [LogPluginV2] or
// [ContainerPluginV2]. Otherwise the Log input is instantiated.
func NewV1Input(
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

// LogPluginV2 returns a v2.Plugin with a manager that can redirect
// a Log input configuration to Filestream via the Redirector interface.
// The Loader resolves the filestream plugin from its own registry;
// this package no longer imports or instantiates filestream directly.
func LogPluginV2(logger *logp.Logger) v2.Plugin {
	return pluginV2(logger, logPluginName)
}

// ContainerPluginV2 returns a v2.Plugin with a manager that can redirect
// a Container input configuration to Filestream via the Redirector interface.
func ContainerPluginV2(logger *logp.Logger) v2.Plugin {
	return pluginV2(logger, containerPluginName)
}

// pluginV2 builds a v2.Plugin whose manager implements Redirector
// for translating log/container configs to filestream.
func pluginV2(logger *logp.Logger, pluginName string) v2.Plugin {
	return v2.Plugin{
		Name:      pluginName,
		Stability: feature.Stable,
		Info:      "log input running filestream",
		Doc:       "Log input running Filestream input",
		Manager:   manager{logger: logger},
	}
}

// manager implements v2.InputManager and v2.Redirector for the
// log and container input types. It delegates to filestream via
// the Loader's plugin registry rather than importing filestream.
type manager struct {
	logger *logp.Logger
}

// Init is a no-op. The filestream manager is initialised by the
// Loader from its own registry; this manager has no resources.
func (m manager) Init(_ unison.Group) error { return nil }

// Create unconditionally returns ErrUnknownInput. When a redirect is
// needed, the Loader handles it via the Redirector interface before
// reaching Create. When no redirect is needed, compat.Combine falls
// through to the V1 log input.
func (m manager) Create(_ *config.C) (v2.Input, error) {
	return nil, v2.ErrUnknownInput
}

// Redirect implements v2.Redirector. It checks whether the config
// should run as filestream and, if so, translates the config and
// returns the target type. The Loader resolves the filestream plugin
// and calls its Create with the translated config.
func (m manager) Redirect(cfg *config.C) (string, *config.C, error) {
	asFilestream, err := runAsFilestream(m.logger, cfg)
	if err != nil {
		return "", nil, err
	}
	if !asFilestream {
		return "", nil, nil
	}

	newCfg, err := convertConfig(m.logger, cfg)
	if err != nil {
		return "", nil, fmt.Errorf("cannot translate log config to filestream: %w", err)
	}

	// runAsFilestream validated that id exists when redirect is active.
	id, _ := cfg.String("id", -1)
	m.logger.Infow("Log input (deprecated) running as Filestream input", "id", id)
	return "filestream", newCfg, nil
}
