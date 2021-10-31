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

package apmhttputil

import (
	"strconv"
	"strings"
)

// ForwardedHeader holds information extracted from a "Forwarded" HTTP header.
type ForwardedHeader struct {
	For   string
	Host  string
	Proto string
}

// ParseForwarded parses a "Forwarded" HTTP header.
func ParseForwarded(f string) ForwardedHeader {
	// We only consider the first value in the sequence,
	// if there are multiple. Disregard everything after
	// the first comma.
	if comma := strings.IndexRune(f, ','); comma != -1 {
		f = f[:comma]
	}
	var result ForwardedHeader
	for f != "" {
		field := f
		if semi := strings.IndexRune(f, ';'); semi != -1 {
			field = f[:semi]
			f = f[semi+1:]
		} else {
			f = ""
		}
		eq := strings.IndexRune(field, '=')
		if eq == -1 {
			// Malformed field, ignore.
			continue
		}
		key := strings.TrimSpace(field[:eq])
		value := strings.TrimSpace(field[eq+1:])
		if len(value) > 0 && value[0] == '"' {
			var err error
			value, err = strconv.Unquote(value)
			if err != nil {
				// Malformed, ignore
				continue
			}
		}
		switch strings.ToLower(key) {
		case "for":
			result.For = value
		case "host":
			result.Host = value
		case "proto":
			result.Proto = value
		}
	}
	return result
}
