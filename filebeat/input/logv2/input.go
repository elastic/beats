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

// runAsFilestream checks whether the configuration should be run as
// Filestream input, on any error the boolean value must be ignore and
// no input started. runAsFilestream also sets the input type accordingly.
func runAsFilestream(cfg *config.C) (bool, error) {
	// Global feature flag that forces all Log input instances
	// to run as Filestream.
	if features.LogInputRunFilestream() {
		return true, nil
	}

	// Only allow to run the Log input as Filestream if Filebeat
	// is running under Elastic Agent.
	if !management.UnderAgent() {
		return false, nil
	}

	// ID is required to run as Filestream input
	if !cfg.HasField("id") {
		return false, nil
	}

	if ok := cfg.HasField("run_as_filestream"); ok {
		runAsFilestream, err := cfg.Bool("run_as_filestream", -1)
		if err != nil {
			return false, fmt.Errorf("cannot parse 'run_as_filestream': %w", err)
		}

		if runAsFilestream {
			if err := cfg.SetString("type", -1, "filestream"); err != nil {
				return false, fmt.Errorf("cannot set 'type': %w", err)
			}

			return true, nil
		}
	}

	return false, nil
}

// newV1Input instantiates the Log input. If 'run_as_filestream' is
// true, then v2.ErrUnknownInput is returned so the Filestream input
// can be instantiated.
func newV1Input(
	cfg *config.C,
	outlet channel.Connector,
	context v1.Context,
	logger *logp.Logger,
) (v1.Input, error) {
	// Inputs V1 should be tried last, so if this function is run we are
	// supposed to be running as the Log input. However do not rely on the
	// factory implementation, also check whether to run as Log or Filestream
	// inputs.
	asFilestream, err := runAsFilestream(cfg)
	if err != nil {
		return nil, err
	}

	if asFilestream {
		return nil, v2.ErrUnknownInput
	}

	inp, err := loginput.NewInput(cfg, outlet, context, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create log input: %w", err)
	}

	logger.Debug("Log input running as Log input")
	return inp, err
}

// PluginV2 returns a v2.Plugin with a manager that checks whether
// the config is from a Log input that should run as Filestream.
// If that is the case the Log input configuration is  converted to
// Filestream and the Filestream input returned.
// Otherwise v2.ErrUnknownInput is returned.
func PluginV2(logger *logp.Logger, store statestore.States) v2.Plugin {
	// The InputManager for Filestream input is from an internal package, so we
	// cannot instantiate it directly here. To circumvent that, we instantiate
	// the whole Filestream Plugin
	filestreamPlugin := filestream.Plugin(logger, store)

	m := manager{
		next:   filestreamPlugin.Manager,
		logger: logger,
	}
	filestreamPlugin.Manager = m

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

func (m manager) Create(cfg *config.C) (v2.Input, error) {
	// When inputs are created, inputs V2 are tried first, so if we
	// are supposed to run as the Log input, return v2.ErrUnknownInput
	asFilestream, err := runAsFilestream(cfg)
	if err != nil {
		return nil, err
	}

	if asFilestream {
		newCfg, err := convertConfig(m.logger, cfg)
		if err != nil {
			return nil, fmt.Errorf("cannot translate log config to filestream: %s", err)
		}

		m.logger.Debug("Log input running as Filestream input")
		return m.next.Create(newCfg)
	}

	return nil, v2.ErrUnknownInput
}
