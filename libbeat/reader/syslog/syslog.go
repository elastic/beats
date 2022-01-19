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
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/reader"
)

//go:generate ragel -Z -G2 -o rfc3164_gen.go parser/parser_rfc3164.rl
//go:generate ragel -Z -G2 -o rfc5424_gen.go parser/parser_rfc5424.rl

var (
	ErrPriorityPart   = errors.New("message has invalid or missing priority part")
	ErrPriority       = errors.New("invalid priority value, expected an integer between 0 and 191")
	ErrTimestamp      = errors.New("message has invalid or missing timestamp")
	ErrHostname       = errors.New("message has invalid or missing hostname")
	ErrVersion        = errors.New("invalid version value, expected an integer between 1 and 999")
	ErrAppName        = errors.New("message has invalid or missing app name")
	ErrProcID         = errors.New("message has invalid or missing proc ID")
	ErrMsgID          = errors.New("message has invalid or missing msg ID")
	ErrStructuredData = errors.New("message has invalid or missing structured data")
	ErrSDID           = errors.New("invalid structured data ID value, expected a value")
	ErrSDIDDuplicated = errors.New("message contains duplicated structured data ID")
	ErrSDParam        = errors.New("invalid structured data parameter, expected a value")
)

// Format defines syslog message formats.
type Format int

const (
	// FormatAuto automatically detects the message format.
	FormatAuto Format = iota
	// FormatRFC3164 expects a message to adhere to RFC 3164.
	FormatRFC3164
	// FormatRFC5424 expects a message to adhere to RFC 5424.
	FormatRFC5424
)

// Unpack will unpack value into a Format.
func (f *Format) Unpack(value string) error {
	switch value {
	case "rfc3164":
		*f = FormatRFC3164
	case "rfc5424":
		*f = FormatRFC5424
	case "auto":
		*f = FormatAuto
	default:
		return fmt.Errorf("invalid format: %q", value)
	}

	return nil
}

// Config stores the configuration for the Parser.
type Config struct {
	// The syslog message format.
	Format Format `config:"format"`
	// The timezone used when enriching timestamps without a time zone.
	TimeZone *cfgtype.Timezone `config:"timezone"`
	// If true, errors will be logged.
	LogErrors bool `config:"log_errors"`
	// If true, errors will be added to the message fields under the error.message field.
	AddErrorKey bool `config:"add_error_key"`
}

// DefaultConfig will return a Config with default values.
func DefaultConfig() Config {
	return Config{
		Format:      FormatAuto,
		TimeZone:    cfgtype.MustNewTimezone("Local"),
		LogErrors:   false,
		AddErrorKey: true,
	}
}

// ParseMessage will parse syslog message data formatted as format into fields. loc is used to enrich
// timestamps that lack a time zone.
func ParseMessage(data string, format Format, loc *time.Location) (common.MapStr, time.Time, error) {
	var m message
	var err error

	switch format {
	case FormatAuto:
		if isRFC5424(data) {
			m, err = parseRFC5424(data)
		} else {
			m, err = parseRFC3164(data, loc)
		}
	case FormatRFC3164:
		m, err = parseRFC3164(data, loc)
	case FormatRFC5424:
		m, err = parseRFC5424(data)
	}
	if err != nil {
		return common.MapStr{}, time.Time{}, err
	}

	return m.fields(data), m.timestamp, nil
}

// Parser is a syslog parser that implements parser.Parser.
type Parser struct {
	cfg    *Config
	reader reader.Reader
	logger *logp.Logger
}

// Close closes this Parser.
func (p *Parser) Close() error {
	return p.reader.Close()
}

// Next reads the next message and parses the syslog message.
func (p *Parser) Next() (reader.Message, error) {
	msg, err := p.reader.Next()
	if err != nil {
		return msg, err
	}

	fields, ts, err := ParseMessage(string(msg.Content), p.cfg.Format, p.cfg.TimeZone.Location())
	if err != nil {
		if p.cfg.LogErrors {
			p.logger.Errorf("Error parsing syslog message: %v", err)
		}
		if p.cfg.AddErrorKey {
			_, _ = fields.Put("error.message", "Error parsing syslog message: "+err.Error())
		}
		msg.AddFields(fields)
		return msg, nil
	}

	textValue, ok := fields["message"]
	if !ok {
		if p.cfg.LogErrors {
			p.logger.Error("Error getting syslog message: message field is missing")
		}
		if p.cfg.AddErrorKey {
			_, _ = fields.Put("error.message", "Error getting syslog message: message field is missing")
		}
		msg.AddFields(fields)
		return msg, nil
	}
	textString, ok := textValue.(string)
	if !ok {
		if p.cfg.LogErrors {
			p.logger.Error("Error getting syslog message: message field is not a string")
		}
		if p.cfg.AddErrorKey {
			_, _ = fields.Put("error.message", "Error getting syslog message: message field is not a string")
		}
		msg.AddFields(fields)
		return msg, nil
	}

	msg.Content = []byte(textString)
	msg.Bytes = len(msg.Content)
	msg.AddFields(fields)
	msg.Ts = ts

	return msg, nil
}

// NewParser creates a new Syslog parser.
func NewParser(r reader.Reader, cfg *Config) *Parser {
	return &Parser{
		cfg:    cfg,
		reader: r,
		logger: logp.NewLogger("reader_syslog"),
	}
}
