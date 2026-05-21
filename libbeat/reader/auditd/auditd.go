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

	auditMsg, err := auparse.ParseLogLine(string(msg.Content))
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
