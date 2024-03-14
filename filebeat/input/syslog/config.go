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

package syslog

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/inputsource"
	"github.com/elastic/beats/filebeat/inputsource/tcp"
	"github.com/elastic/beats/filebeat/inputsource/udp"
	"github.com/elastic/beats/libbeat/common"
)

// ConditionConfig : 用于条件表达式，目前支持=、!=、eq、neq、include、exclude、regex、nregex
type ConditionConfig struct {
	Key     string `config:"key"`
	Op      string `config:"op"`
	Value   string `config:"value"`
	matcher MatchFunc
}

func (c *ConditionConfig) GetMatcher() MatchFunc {
	return c.matcher
}

type FilterConfig struct {
	Conditions []ConditionConfig `config:"conditions"`
}

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	Protocol                  common.ConfigNamespace `config:"protocol"`
	SyslogFilters             []FilterConfig         `config:"syslog_filters"`
}

var defaultConfig = config{
	ForwarderConfig: harvester.ForwarderConfig{
		Type: "syslog",
	},
}

type syslogTCP struct {
	tcp.Config    `config:",inline"`
	LineDelimiter string `config:"line_delimiter" validate:"nonzero"`
}

var defaultTCP = syslogTCP{
	Config: tcp.Config{
		Timeout:        time.Minute * 5,
		MaxMessageSize: 20 * humanize.MiByte,
	},
	LineDelimiter: "\n",
}

var defaultUDP = udp.Config{
	MaxMessageSize: 10 * humanize.KiByte,
	Timeout:        time.Minute * 5,
}

func factory(
	cb inputsource.NetworkFunc,
	config common.ConfigNamespace,
) (inputsource.Network, error) {
	n, cfg := config.Name(), config.Config()

	switch n {
	case tcp.Name:
		config := defaultTCP
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}

		splitFunc := tcp.SplitFunc([]byte(config.LineDelimiter))
		if splitFunc == nil {
			return nil, fmt.Errorf("error creating splitFunc from delimiter %s", config.LineDelimiter)
		}

		return tcp.New(&config.Config, splitFunc, cb)
	case udp.Name:
		config := defaultUDP
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
		return udp.New(&config, cb), nil
	default:
		return nil, fmt.Errorf("you must choose between TCP or UDP")
	}
}
