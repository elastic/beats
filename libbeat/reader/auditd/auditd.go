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

//go:build linux

package auditd

import (
	"strconv"
	"strings"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-libaudit/v2/auparse"
)

// Parser parses each line of an audit.log file using go-libaudit's auparse
// package and populates auditd.log.* fields on the message, replacing the
// grok/KV stages in the ingest pipeline.
type Parser struct {
	cfg    Config
	reader reader.Reader
	logger *logp.Logger
}

// NewParser creates a new auditd Parser.
func NewParser(r reader.Reader, cfg Config, logger *logp.Logger) *Parser {
	return &Parser{
		cfg:    cfg,
		reader: r,
		logger: logger.Named("reader_auditd"),
	}
}

// Close closes the underlying reader.
func (p *Parser) Close() error {
	return p.reader.Close()
}

// Next reads the next line and parses it as an auditd log record, populating
// auditd.log.* fields and setting the message timestamp. If a line cannot be
// parsed, it is passed through unchanged (subject to the error config flags).
func (p *Parser) Next() (reader.Message, error) {
	msg, err := p.reader.Next()
	if err != nil {
		return msg, err
	}

	line, nodeVal := stripNodePrefix(string(msg.Content))
	auditMsg, err := auparse.ParseLogLine(line)
	if err != nil {
		if p.cfg.LogErrors {
			p.logger.Errorf("error parsing auditd log line: %v", err)
		}
		if p.cfg.AddErrorKey {
			msg.AddFields(mapstr.M{
				"error": mapstr.M{"message": "error parsing auditd log line: " + err.Error()},
			})
		}
		return msg, nil
	}

	msg.Ts = auditMsg.Timestamp

	logFields := mapstr.M{
		"record_type": auditMsg.RecordType.String(),
		"sequence":    strconv.FormatUint(uint64(auditMsg.Sequence), 10),
	}

	data, dataErr := auditMsg.Data()
	for k, v := range data {
		logFields[k] = v
	}
	if nodeVal != "" {
		logFields["node"] = nodeVal
	}
	// auparse normalises res/success → result, but the ingest pipeline
	// renames auditd.log.res to event.outcome, so restore the alias.
	if result, ok := logFields["result"]; ok {
		logFields["res"] = result
	}
	if dataErr != nil {
		if p.cfg.LogErrors {
			p.logger.Errorf("error extracting auditd data fields: %v", dataErr)
		}
		if p.cfg.AddErrorKey {
			msg.AddFields(mapstr.M{
				"error": mapstr.M{"message": "error extracting auditd data fields: " + dataErr.Error()},
			})
		}
	}

	msg.AddFields(mapstr.M{"auditd": mapstr.M{"log": logFields}})

	return msg, nil
}

// stripNodePrefix removes the "node=<value> " prefix that userspace auditd
// prepends when name_format=hostname is set in auditd.conf. auparse.ParseLogLine
// only handles lines that start with "type=", so the prefix must be stripped
// before parsing. The extracted node value (empty string if absent) is returned.
func stripNodePrefix(line string) (string, string) {
	const prefix = "node="
	if !strings.HasPrefix(line, prefix) {
		return line, ""
	}
	i := strings.IndexByte(line, ' ')
	if i < 0 {
		return line, ""
	}
	return line[i+1:], line[len(prefix):i]
}
