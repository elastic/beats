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

package parser

import (
	"errors"
	"fmt"
	"io"

	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/cfgtype"
	"github.com/elastic/beats/v8/libbeat/reader"
	"github.com/elastic/beats/v8/libbeat/reader/multiline"
	"github.com/elastic/beats/v8/libbeat/reader/readfile"
	"github.com/elastic/beats/v8/libbeat/reader/readjson"
	"github.com/elastic/beats/v8/libbeat/reader/syslog"
)

var (
	ErrNoSuchParser = errors.New("no such parser")
)

// parser transforms or translates the Content attribute of a Message.
// They are able to aggregate two or more Messages into a single one.
type Parser interface {
	io.Closer
	Next() (reader.Message, error)
}

type CommonConfig struct {
	MaxBytes       cfgtype.ByteSize        `config:"max_bytes"`
	LineTerminator readfile.LineTerminator `config:"line_terminator"`
}

type Config struct {
	Suffix string

	pCfg    CommonConfig
	parsers []common.ConfigNamespace
}

func (c *Config) Unpack(cc *common.Config) error {
	tmp := struct {
		Common  CommonConfig             `config:",inline"`
		Parsers []common.ConfigNamespace `config:"parsers"`
	}{
		CommonConfig{
			MaxBytes:       10 * humanize.MiByte,
			LineTerminator: readfile.AutoLineTerminator,
		},
		nil,
	}
	err := cc.Unpack(&tmp)
	if err != nil {
		return err
	}

	newC, err := NewConfig(tmp.Common, tmp.Parsers)
	if err != nil {
		return err
	}
	*c = *newC

	return nil
}

func NewConfig(pCfg CommonConfig, parsers []common.ConfigNamespace) (*Config, error) {
	var suffix string
	for _, ns := range parsers {
		name := ns.Name()
		switch name {
		case "multiline":
			var config multiline.Config
			cfg := ns.Config()
			err := cfg.Unpack(&config)
			if err != nil {
				return nil, fmt.Errorf("error while parsing multiline parser config: %+v", err)
			}
		case "ndjson":
			var config readjson.ParserConfig
			cfg := ns.Config()
			err := cfg.Unpack(&config)
			if err != nil {
				return nil, fmt.Errorf("error while parsing ndjson parser config: %+v", err)
			}
		case "container":
			config := readjson.DefaultContainerConfig()
			cfg := ns.Config()
			err := cfg.Unpack(&config)
			if err != nil {
				return nil, fmt.Errorf("error while parsing container parser config: %+v", err)
			}
			if config.Stream != readjson.All {
				if suffix != "" {
					return nil, fmt.Errorf("only one stream selection is allowed")
				}
				suffix = config.Stream.String()
			}
		case "syslog":
			config := syslog.DefaultConfig()
			cfg := ns.Config()
			err := cfg.Unpack(&config)
			if err != nil {
				return nil, fmt.Errorf("error while parsing syslog parser config: %w", err)
			}
		default:
			return nil, fmt.Errorf("%s: %s", ErrNoSuchParser, name)
		}
	}

	return &Config{
		Suffix:  suffix,
		pCfg:    pCfg,
		parsers: parsers,
	}, nil

}

func (c *Config) Create(in reader.Reader) Parser {
	p := in
	for _, ns := range c.parsers {
		name := ns.Name()
		switch name {
		case "multiline":
			var config multiline.Config
			cfg := ns.Config()
			err := cfg.Unpack(&config)
			if err != nil {
				return p
			}
			p, err = multiline.New(p, "\n", int(c.pCfg.MaxBytes), &config)
			if err != nil {
				return p
			}
		case "ndjson":
			var config readjson.ParserConfig
			cfg := ns.Config()
			err := cfg.Unpack(&config)
			if err != nil {
				return p
			}
			p = readjson.NewJSONParser(p, &config)
		case "container":
			config := readjson.DefaultContainerConfig()
			cfg := ns.Config()
			err := cfg.Unpack(&config)
			if err != nil {
				return p
			}
			p = readjson.NewContainerParser(p, &config)
		case "syslog":
			config := syslog.DefaultConfig()
			cfg := ns.Config()
			err := cfg.Unpack(&config)
			if err != nil {
				return p
			}
			p = syslog.NewParser(p, &config)
		default:
			return p
		}
	}

	return p
}
