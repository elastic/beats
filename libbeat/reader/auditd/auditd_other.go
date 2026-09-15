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
	"time"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Parser wraps the internal auditd parsing implementation. On non-Linux
// platforms it acts as a pass-through.
type Parser struct {
	r reader.Reader
}

// NewParser is a pass-through on non-Linux platforms. The auditd parser
// requires go-libaudit which is Linux-only.
func NewParser(r reader.Reader, _ Config, _ *logp.Logger) *Parser {
	return &Parser{r: r}
}

func (p *Parser) Next() (reader.Message, error) { return p.r.Next() }

func (p *Parser) SetReadDeadline(t time.Time) bool {
	return reader.SetReadDeadline(p.r, t)
}

func (p *Parser) Close() error { return p.r.Close() }
