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

package configure

import (
	"flag"
	"fmt"
	"strings"

	"go.uber.org/zap/zapcore"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// CLI flags for configuring logging.
var (
	verbose        bool
	toStderr       bool
	debugSelectors []string
	environment    logp.Environment
)

type environmentVar logp.Environment

func init() {
	flag.BoolVar(&verbose, "v", false, "Log at INFO level")
	flag.BoolVar(&toStderr, "e", false, "Log to stderr and disable syslog/file output")
	config.StringArrVarFlag(nil, &debugSelectors, "d", "Enable certain debug selectors")
	flag.Var((*environmentVar)(&environment), "environment", "set environment being ran in")
}

func GetEnvironment() logp.Environment {
	return environment
}

// Logging builds a logp.Config based on the given common.Config and the specified
// CLI flags.
func Logging(beatName string, cfg *config.C) error {
	config := logp.DefaultConfig(environment)
	config.Beat = beatName
	if cfg != nil {
		if err := cfg.Unpack(&config); err != nil {
			return err
		}
	}

	applyFlags(&config)
	return logp.Configure(config)
}

// LoggingWithOutputs builds a logp.Config based on the given common.Config and the specified
// CLI flags along with the given outputs.
func LoggingWithOutputs(beatName string, cfg *config.C, outputs ...zapcore.Core) error {
	config := logp.DefaultConfig(environment)
	config.Beat = beatName
	if cfg != nil {
		if err := cfg.Unpack(&config); err != nil {
			return err
		}
	}

	applyFlags(&config)
	return logp.ConfigureWithOutputs(config, outputs...)
}

// LoggingWithTypedOutputs applies some defaults then calls ConfigureWithTypedOutputs
//
// Deprecated: Prefer using localized loggers. Use logp.LoggingWithTypedOutputsLocal.
func LoggingWithTypedOutputs(beatName string, cfg, typedCfg *config.C, logKey, kind string, outputs ...zapcore.Core) error {
	config := logp.DefaultConfig(environment)
	config.Beat = beatName
	if cfg != nil {
		if err := cfg.Unpack(&config); err != nil {
			return err
		}
	}

	applyFlags(&config)

	typedLogpConfig := logp.DefaultEventConfig(environment)
	defaultName := typedLogpConfig.Files.Name
	typedLogpConfig.Beat = beatName
	if typedCfg != nil {
		if err := typedCfg.Unpack(&typedLogpConfig); err != nil {
			return fmt.Errorf("cannot unpack typed output config: %w", err)
		}
	}

	// Make sure we're always running on the same log level
	typedLogpConfig.Level = config.Level
	typedLogpConfig.Selectors = config.Selectors

	// If the name has not been configured, make it {beatName}-events-data
	if typedLogpConfig.Files.Name == defaultName {
		typedLogpConfig.Files.Name = beatName + "-events-data"
	}

	return logp.ConfigureWithTypedOutput(config, typedLogpConfig, logKey, kind, outputs...)
}

// LoggingWithTypedOutputs applies some defaults and returns a logger instance
func LoggingWithTypedOutputsLocal(beatName string, cfg, typedCfg *config.C, logKey, kind string, outputs ...zapcore.Core) (*logp.Logger, error) {
	config := logp.DefaultConfig(environment)
	config.Beat = beatName
	if cfg != nil {
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
	}

	applyFlags(&config)

	typedLogpConfig := logp.DefaultEventConfig(environment)
	defaultName := typedLogpConfig.Files.Name
	typedLogpConfig.Beat = beatName
	if typedCfg != nil {
		if err := typedCfg.Unpack(&typedLogpConfig); err != nil {
			return nil, fmt.Errorf("cannot unpack typed output config: %w", err)
		}
	}

	// Make sure we're always running on the same log level
	typedLogpConfig.Level = config.Level
	typedLogpConfig.Selectors = config.Selectors

	// If the name has not been configured, make it {beatName}-events-data
	if typedLogpConfig.Files.Name == defaultName {
		typedLogpConfig.Files.Name = beatName + "-events-data"
	}

	return logp.ConfigureWithTypedOutputLocal(config, typedLogpConfig, logKey, kind, outputs...)
}

func applyFlags(cfg *logp.Config) {
	if toStderr {
		cfg.ToStderr = true
	}
	if cfg.Level > logp.InfoLevel && verbose {
		cfg.Level = logp.InfoLevel
	}
	for _, selectors := range debugSelectors {
		cfg.Selectors = append(cfg.Selectors, strings.Split(selectors, ",")...)
	}

	// Elevate level if selectors are specified on the CLI.
	if len(debugSelectors) > 0 {
		cfg.Level = logp.DebugLevel
	}

	// fix ToFiles and ToStderr
	if cfg.ToFiles && cfg.ToStderr {
		// This scenario is possible when user has selected an environment and has manually set a new logging config.
		// The resulting config i.e. defaultConfig + userConfig can have both ToFiles and ToStderr enabled.
		// We can't have both ToFiles and ToStderr enabled, so we need to fix this with some additional steps.

		defaultConfig := logp.DefaultConfig(environment)

		// We'll use the default configuration to decide which option should be active.
		if defaultConfig.ToFiles != cfg.ToFiles {
			// The User has enabled ToFiles in the config. So, that takes precedence.
			cfg.ToStderr = !cfg.ToFiles
		} else if defaultConfig.ToStderr != cfg.ToStderr {
			// The User has enabled ToStderr in the config. So, that takes precedence.
			cfg.ToFiles = !cfg.ToStderr
		}
	}
}

func (v *environmentVar) Set(in string) error {
	env := logp.ParseEnvironment(in)
	if env == logp.InvalidEnvironment {
		return fmt.Errorf("'%v' is not supported", in)
	}

	*(*logp.Environment)(v) = env
	return nil
}

func (v *environmentVar) String() string {
	return (*logp.Environment)(v).String()
}
