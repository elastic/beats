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

//go:build !linux

package auditd

import (
	"errors"
	"time"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Stub for non-Linux builds
type Parser struct{}

func (p *Parser) Close() error { return nil }

func (p *Parser) Next() (reader.Message, error) {
	return reader.Message{}, errors.New("auditd parser is not supported on this platform")
}

func NewParser(_ reader.Reader, _ Config, _ *logp.Logger) *Parser {
	return &Parser{}
}

// SetReadDeadline is a no-op on platforms without auditd support.
func (p *Parser) SetReadDeadline(time.Time) bool { return false }
