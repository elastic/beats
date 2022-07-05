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
	"strings"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// stringToInt converts a string, assumed to be ASCII numeric characters, to an int.
// This is a simplified version of the fast path for strconv.Atoi, with error handling removed.
func stringToInt(v string) int {
	var n int

	if len(v) == 0 {
		return 0
	}

	s := v
	if v[0] == '-' || v[0] == '+' {
		if len(v) == 1 {
			return 0
		}
		s = s[1:]
	}

	for _, ch := range s {
		n = n*10 + int(ch-'0')
	}
	if v[0] == '-' {
		n = -n
	}

	return n
}

// removeBytes will remove bytes at the given positions in a string. An offset may be given
// to adjust the indexes of the positions (useful in cases where value is a substring
// of a larger string). Note that this function does not operate at the rune level. Removing
// bytes arbitrarily from a string may result in an invalid UTF-8 string if bytes are removed
// from a multibyte rune sequence. If any of the following are true, the original string is returned:
//    - positions is empty
//    - offset is less than 0
//    - length of value is less than length of positions.
//    - values of positions (including if they are offset) yield a result less than zero or greater
//    than or equal to the length of value (invalid slice index operation)
func removeBytes(value string, positions []int, offset int) string {
	var sb strings.Builder
	var tok int

	// If no positions are provided, return original string to avoid allocation.
	if len(positions) == 0 {
		return value
	}
	// Check bounds of inputs.
	if offset < 0 || len(value) < len(positions) {
		return value
	}
	// Check bounds of positions.
	for _, pos := range positions {
		if pos-offset < 0 || pos-offset >= len(value) {
			return value
		}
	}

	sb.Grow(len(value) - len(positions))
	for _, pos := range positions {
		_, _ = sb.WriteString(value[tok : pos-offset])
		tok = pos - offset + 1
	}
	_, _ = sb.WriteString(value[tok:])

	return sb.String()
}

// mapIndexToString will return the value at the index of idx in values. It returns the value
// and true if the index is valid, otherwise and empty string and false if the index is invalid.
func mapIndexToString(idx int, values []string) (string, bool) {
	if idx < 0 || idx >= len(values) {
		return "", false
	}

	return values[idx], true
}

// appendStringField appends value to field. If field is nil (not present in the map), then
// the resulting field value will be a string. If the existing field is a string, then field
// value will be converted to a string slice. If the existing field is a string slice or
// interface slice, then the new value will be appended. If the existing value is some
// other type, then this function does nothing.
func appendStringField(m mapstr.M, field, value string) {
	v, _ := m.GetValue(field)
	switch t := v.(type) {
	case nil:
		_, _ = m.Put(field, value)
	case string:
		_, _ = m.Put(field, []string{t, value})
	case []string:
		_, _ = m.Put(field, append(t, value))
	case []interface{}:
		_, _ = m.Put(field, append(t, value))
	}
}

// joinStr joins strings a and b using separator sep. If a and b are empty, the
// result is an empty string. If a or b are empty, the other is returned.
func joinStr(a, b, sep string) string {
	if a == "" && b == "" {
		return ""
	}
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}

	return a + sep + b
}
