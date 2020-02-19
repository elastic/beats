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

package configutil

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Size represents a size in bytes.
type Size int64

// Common power-of-two sizes.
const (
	Byte  Size = 1
	KByte Size = 1024
	MByte Size = 1024 * 1024
	GByte Size = 1024 * 1024 * 1024
)

// Bytes returns s as a number of bytes.
func (s Size) Bytes() int64 {
	return int64(s)
}

// String returns s in its most compact string representation.
func (s Size) String() string {
	if s == 0 {
		return "0B"
	}
	switch {
	case s%GByte == 0:
		return fmt.Sprintf("%dGB", s/GByte)
	case s%MByte == 0:
		return fmt.Sprintf("%dMB", s/MByte)
	case s%KByte == 0:
		return fmt.Sprintf("%dKB", s/KByte)
	default:
		return fmt.Sprintf("%dB", s)
	}
}

// ParseSize parses s as a size, in bytes.
//
// Valid size units are "b", "kb", "mb", "gb".
func ParseSize(s string) (Size, error) {
	orig := s
	var mul Size = 1
	if strings.HasPrefix(s, "-") {
		mul = -1
		s = s[1:]
	}

	sep := -1
	for i, c := range s {
		if sep == -1 {
			if c < '0' || c > '9' {
				sep = i
				break
			}
		}
	}
	if sep == -1 {
		return 0, fmt.Errorf("missing unit in size %s (allowed units: B, KB, MB, GB)", orig)
	}

	n, err := strconv.ParseInt(s[:sep], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid size %s", orig)
	}
	switch strings.ToLower(s[sep:]) {
	case "gb":
		mul = GByte
	case "mb":
		mul = MByte
	case "kb":
		mul = KByte
	case "b":
	default:
		for _, c := range s[sep:] {
			if unicode.IsSpace(c) {
				return 0, fmt.Errorf("invalid character %q in size %s", c, orig)
			}
		}
		return 0, fmt.Errorf("invalid unit in size %s (allowed units: B, KB, MB, GB)", orig)
	}
	return mul * Size(n), nil
}
