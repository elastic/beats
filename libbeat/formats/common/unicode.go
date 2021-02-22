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
	"encoding/binary"
	"unicode/utf16"
)

// ReadUnicode decodes a unicode string ending with a null
func ReadUnicode(data []byte, offset int) string {
	encode := []uint16{}
	for {
		if len(data) < offset+1 {
			return string(utf16.Decode(encode))
		}
		value := binary.LittleEndian.Uint16(data[offset : offset+2])
		if value == 0 {
			return string(utf16.Decode(encode))
		}
		encode = append(encode, value)
		offset += 2
	}
}
