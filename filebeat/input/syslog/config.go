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

	"github.com/menderesk/beats/v7/filebeat/harvester"
	"github.com/menderesk/beats/v7/filebeat/inputsource"
	"github.com/menderesk/beats/v7/filebeat/inputsource/common/streaming"
	"github.com/menderesk/beats/v7/filebeat/inputsource/tcp"
	"github.com/menderesk/beats/v7/filebeat/inputsource/udp"
	"github.com/menderesk/beats/v7/filebeat/inputsource/unix"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/cfgtype"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	Format                    syslogFormat           `config:"format"`
	Protocol                  common.ConfigNamespace `config:"protocol"`
	Timezone                  *cfgtype.Timezone      `config:"timezone"`
}

type syslogFormat int

const (
	syslogFormatRFC3164 = iota
	syslogFormatRFC5424
	syslogFormatAuto
)

var syslogFormats = map[string]syslogFormat{
	"rfc3164": syslogFormatRFC3164,
	"rfc5424": syslogFormatRFC5424,
	"auto":    syslogFormatAuto,
}

var defaultConfig = config{
	ForwarderConfig: harvester.ForwarderConfig{
		Type: "syslog",
	},
	Format:   syslogFormatRFC3164,
	Timezone: cfgtype.MustNewTimezone("Local"),
}

type syslogTCP struct {
	tcp.Config    `config:",inline"`
	LineDelimiter string                `config:"line_delimiter" validate:"nonzero"`
	Framing       streaming.FramingType `config:"framing"`
}

var defaultTCP = syslogTCP{
	Config: tcp.Config{
		Timeout:        time.Minute * 5,
		MaxMessageSize: 20 * humanize.MiByte,
	},
	LineDelimiter: "\n",
}

type syslogUnix struct {
	unix.Config `config:",inline"`
}

func defaultUnix() syslogUnix {
	return syslogUnix{
		Config: unix.Config{
			Timeout:        time.Minute * 5,
			MaxMessageSize: 20 * humanize.MiByte,
			LineDelimiter:  "\n",
		},
	}
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

		splitFunc, err := streaming.SplitFunc(config.Framing, []byte(config.LineDelimiter))
		if err != nil {
			return nil, err
		}

		logger := logp.NewLogger("input.syslog.tcp").With("address", config.Config.Host)
		factory := streaming.SplitHandlerFactory(inputsource.FamilyTCP, logger, tcp.MetadataCallback, nf, splitFunc)

		return tcp.New(&config.Config, factory)
	case unix.Name:
		config := defaultUnix()
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}

		logger := logp.NewLogger("input.syslog.unix").With("path", config.Config.Path)

		return unix.New(logger, &config.Config, nf)

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

func (f *syslogFormat) Unpack(value string) error {
	format, ok := syslogFormats[value]
	if !ok {
		return fmt.Errorf("invalid format '%s'", value)
	}
	*f = format
	return nil
}
