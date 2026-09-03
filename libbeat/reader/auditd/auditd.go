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
	"regexp"
	"strconv"
	"strings"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-libaudit/v2/auparse"
)

// Parser wraps the internal auditd parsing implementation selected by
// Config.Mode. It satisfies reader.Reader and reader.DeadlineSetter.
type Parser struct {
	r reader.Reader
}

// NewParser creates a new auditd Parser. The behaviour is controlled by
// cfg.Mode: ModeCoalesce groups records by sequence number into compound
// events, ModeParse parses each line independently, and ModeNone passes
// the input through unchanged.
func NewParser(r reader.Reader, cfg Config, logger *logp.Logger) *Parser {
	switch cfg.Mode {
	case ModeCoalesce:
		return &Parser{r: newCoalescingParser(r, cfg, logger)}
	case ModeParse:
		return &Parser{r: newParseParser(r, cfg, logger)}
	default:
		return &Parser{r: r}
	}
}

func (p *Parser) Next() (reader.Message, error) { return p.r.Next() }

// SetReadDeadline delegates to the wrapped reader (see reader.DeadlineSetter).
func (p *Parser) SetReadDeadline(t time.Time) bool {
	return reader.SetReadDeadline(p.r, t)
}

func (p *Parser) Close() error { return p.r.Close() }

// parseParser parses each line of an audit.log file using go-libaudit's auparse
// package and populates auditd.log.* fields on the message.
type parseParser struct {
	cfg    Config
	reader reader.Reader
	logger *logp.Logger
}

func newParseParser(r reader.Reader, cfg Config, logger *logp.Logger) *parseParser {
	return &parseParser{
		cfg:    cfg,
		reader: r,
		logger: logger.Named("reader_auditd"),
	}
}

// Next reads the next line and parses it as an auditd log record, populating
// auditd.log.* fields and setting the message timestamp. If a line cannot be
// parsed, it is passed through unchanged (subject to the error config flags).
func (p *parseParser) Next() (reader.Message, error) {
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
	for k, v := range data {
		logFields[k] = v
	}
	if nodeVal != "" {
		logFields["node"] = nodeVal
	}

	// auparse moves the audit rule key(s) from the data map to AuditMessage.Tags.
	// Restore them as auditd.log.key so the parsed event keeps the rule key.
	if tags, _ := auditMsg.Tags(); len(tags) > 0 {
		if len(tags) == 1 {
			logFields["key"] = tags[0]
		} else {
			logFields["key"] = tags
		}
	}

	// Re-parse the inner msg='...' block to recover multi-word unquoted values
	// that auparse truncates at the first whitespace. We only overwrite a field
	// when the raw value is strictly longer and the current value is a prefix
	// of it, which identifies truncation while preserving auparse enrichments
	// (e.g. syscall number resolved to a name).
	if m := innerMsgRe.FindStringSubmatch(auditMsg.RawData); len(m) > 1 {
		for _, kv := range innerMsgKVRe.FindAllStringSubmatch(m[1], -1) {
			k, rawV := kv[1], strings.Trim(kv[2], `'"`)
			if existing, ok := logFields[k]; ok {
				if s, ok := existing.(string); ok && len(rawV) > len(s) && strings.HasPrefix(rawV, s) {
					logFields[k] = rawV
				}
			}
		}
	}
	// Extract avc.action and avc.request from SELinux AVC records.
	if m := avcRe.FindStringSubmatch(auditMsg.RawData); len(m) > 2 {
		logFields["avc"] = mapstr.M{
			"action":  m[1],
			"request": m[2],
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
<<<<<<< HEAD
=======

var (
	innerMsgRe = regexp.MustCompile(`\bmsg='([^']*)'`)

	// innerMsgKVRe extracts individual key=value pairs from inner msg content.
	// Unquoted values may span multiple words (e.g. op=adding group to /etc/group)
	// because auparse's KV regex stops at the first whitespace. The boundary is
	// the next key=value token.
	innerMsgKVRe = regexp.MustCompile(`([a-z][a-z0-9_-]*)=(.*?)(?:\s+[a-z][a-z0-9_-]+=|\s*$)`)

	avcRe = regexp.MustCompile(`\bavc:\s+(\w+)\s+\{\s+(\w+)\s+\}\s+for\s+`)
)

// SetReadDeadline delegates to the wrapped reader (see reader.DeadlineSetter).
func (p *parseParser) SetReadDeadline(t time.Time) bool {
	return reader.SetReadDeadline(p.reader, t)
}

func (p *parseParser) Close() error {
	return p.reader.Close()
}
>>>>>>> f449761 (libbeat/reader/auditd: add multi-record coalescing mode (#52648))
