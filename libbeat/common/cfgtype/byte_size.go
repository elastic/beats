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

package cfgtype

import (
	"unicode"

	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
)

// ByteSize defines a new configuration option that will parse `go-humanize` compatible values into a
// int64 when the suffix is valid or will fallback to bytes.
type ByteSize int64

// Unpack converts a size defined from a human readable format into bytes.
func (s *ByteSize) Unpack(v string) error {
	sz, err := humanize.ParseBytes(v)
	if isRawBytes(v) {
		cfgwarn.Deprecate("7.0.0", "size now requires a unit (KiB, MiB, etc...), current value: %s.", v)
	}
	if err != nil {
		return err
	}

	*s = ByteSize(sz)
	return nil
}

func isRawBytes(v string) bool {
	for _, c := range v {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}
