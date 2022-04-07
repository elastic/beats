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

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
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
	common.StringArrVarFlag(nil, &debugSelectors, "d", "Enable certain debug selectors")
	flag.Var((*environmentVar)(&environment), "environment", "set environment being ran in")
}

// Logging builds a logp.Config based on the given common.Config and the specified
// CLI flags.
func Logging(beatName string, cfg *common.Config) error {
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
func LoggingWithOutputs(beatName string, cfg *common.Config, outputs ...zapcore.Core) error {
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
