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

// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/processors/util"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type LocationWithID struct {
	ID  string         `config:"id"`
	Geo util.GeoConfig `config:"geo"`
}

// Config defines the structure of heartbeat.yml.
type Config struct {
	RunOnce        bool                 `config:"run_once"`
	Monitors       []*conf.C            `config:"monitors"`
	ConfigMonitors *conf.C              `config:"config.monitors"`
	Scheduler      Scheduler            `config:"scheduler"`
	Autodiscover   *autodiscover.Config `config:"autodiscover"`
	Jobs           map[string]*JobLimit `config:"jobs"`
	RunFrom        *LocationWithID      `config:"run_from"`
	SocketTrace    *SocketTrace         `config:"socket_trace"`
}

type JobLimit struct {
	Limit int64 `config:"limit" validate:"min=0"`
}

// Scheduler defines the syntax of a heartbeat.yml scheduler block.
type Scheduler struct {
	Limit    int64  `config:"limit"  validate:"min=0"`
	Location string `config:"location"`
}

// DefaultConfig is the canonical instantiation of Config.
func DefaultConfig() *Config {
	limits := map[string]*JobLimit{
		"browser": {Limit: 2},
	}

	// Read the env key SYNTHETICS_LIMIT_{TYPE} for each type of monitor to set scaling limits
	// hard coded list of types to avoid cycles in current plugin system.
	// TODO: refactor plugin system to DRY this up
	for _, t := range []string{"http", "tcp", "icmp", "browser"} {
		envKey := fmt.Sprintf("SYNTHETICS_LIMIT_%s", strings.ToUpper(t))
		if limitStr := os.Getenv(envKey); limitStr != "" {
			tLimitVal, err := strconv.ParseInt(limitStr, 10, 64)
			if err != nil {
				logp.L().Warnf("Could not parse job limit env var %s with value '%s' as integer", envKey, limitStr)
				continue
			}

			limits[t] = &JobLimit{Limit: tLimitVal}
		}
	}

	return &Config{
		Jobs: limits,
	}
}

type SocketTrace struct {
	Path string        `config:"path"`
	Wait time.Duration `config:"wait"`
}
