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

	"github.com/elastic/beats/v7/filebeat/harvester"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	netcommon "github.com/elastic/beats/v7/filebeat/inputsource/common"
	"github.com/elastic/beats/v7/filebeat/inputsource/tcp"
	"github.com/elastic/beats/v7/filebeat/inputsource/udp"
	"github.com/elastic/beats/v7/filebeat/inputsource/unix"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	Protocol                  common.ConfigNamespace `config:"protocol"`
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

type syslogUnix struct {
	unix.Config   `config:",inline"`
	LineDelimiter string `config:"line_delimiter" validate:"nonzero"`
}

var defaultUnix = syslogUnix{
	Config: unix.Config{
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
	nf inputsource.NetworkFunc,
	config common.ConfigNamespace,
) (inputsource.Network, error) {
	n, cfg := config.Name(), config.Config()

	switch n {
	case tcp.Name:
		config := defaultTCP
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}

		splitFunc := netcommon.SplitFunc([]byte(config.LineDelimiter))
		if splitFunc == nil {
			return nil, fmt.Errorf("error creating splitFunc from delimiter %s", config.LineDelimiter)
		}

		logger := logp.NewLogger("input.syslog.tcp").With("address", config.Config.Host)
		factory := netcommon.SplitHandlerFactory(netcommon.FamilyTCP, logger, tcp.MetadataCallback, nf, splitFunc)

		return tcp.New(&config.Config, factory)
	case unix.Name:
		cfgwarn.Beta("Syslog Unix socket support is beta.")

		config := defaultUnix
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}

		splitFunc := netcommon.SplitFunc([]byte(config.LineDelimiter))
		if splitFunc == nil {
			return nil, fmt.Errorf("error creating splitFunc from delimiter %s", config.LineDelimiter)
		}

		logger := logp.NewLogger("input.syslog.unix").With("path", config.Config.Path)
		factory := netcommon.SplitHandlerFactory(netcommon.FamilyUnix, logger, unix.MetadataCallback, nf, splitFunc)

		return unix.New(&config.Config, factory)

	case udp.Name:
		config := defaultUDP
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
		return udp.New(&config, nf), nil
	default:
		return nil, fmt.Errorf("you must choose between TCP or UDP")
	}
}
