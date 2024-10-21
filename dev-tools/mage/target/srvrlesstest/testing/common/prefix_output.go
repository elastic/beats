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

package common

import (
	"bytes"
	"strings"
)

// prefixOutput is an `io.Writer` that prefixes each written line with the provided prefix text
type prefixOutput struct {
	logger    Logger
	prefix    string
	remainder []byte
}

// NewPrefixOutput creates a prefix output `io.Writer`.
func NewPrefixOutput(logger Logger, prefix string) *prefixOutput {
	return &prefixOutput{
		logger: logger,
		prefix: prefix,
	}
}

func (r *prefixOutput) Write(p []byte) (int, error) {
	if len(p) == 0 {
		// nothing to do
		return 0, nil
	}
	offset := 0
	for {
		idx := bytes.IndexByte(p[offset:], '\n')
		if idx < 0 {
			// not all used add to remainder to be used on next call
			r.remainder = append(r.remainder, p[offset:]...)
			return len(p), nil
		}

		var line []byte
		if r.remainder != nil {
			line = r.remainder
			r.remainder = nil
			line = append(line, p[offset:offset+idx]...)
		} else {
			line = append(line, p[offset:offset+idx]...)
		}
		offset += idx + 1
		// drop '\r' from line (needed for Windows)
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[0 : len(line)-1]
		}
		if len(line) == 0 {
			// empty line
			continue
		}
		str := strings.TrimSpace(string(line))
		r.logger.Logf("%s%s", r.prefix, str)
	}
}
