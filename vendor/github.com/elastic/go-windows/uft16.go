// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package windows

import (
	"fmt"
	"unicode/utf16"
)

// UTF16BytesToString returns a string that is decoded from the UTF-16 bytes.
// The byte slice must be of even length otherwise an error will be returned.
// The integer returned is the offset to the start of the next string with
// buffer if it exists, otherwise -1 is returned.
func UTF16BytesToString(b []byte) (string, int, error) {
	if len(b)%2 != 0 {
		return "", 0, fmt.Errorf("slice must have an even length (length=%d)", len(b))
	}

	offset := -1

	// Find the null terminator if it exists and re-slice the b.
	if nullIndex := indexNullTerminator(b); nullIndex > -1 {
		if len(b) > nullIndex+2 {
			offset = nullIndex + 2
		}

		b = b[:nullIndex]
	}

	s := make([]uint16, len(b)/2)
	for i := range s {
		s[i] = uint16(b[i*2]) + uint16(b[(i*2)+1])<<8
	}

	return string(utf16.Decode(s)), offset, nil
}

// indexNullTerminator returns the index of a null terminator within a buffer
// containing UTF-16 encoded data. If the null terminator is not found -1 is
// returned.
func indexNullTerminator(b []byte) int {
	if len(b) < 2 {
		return -1
	}

	for i := 0; i < len(b); i += 2 {
		if b[i] == 0 && b[i+1] == 0 {
			return i
		}
	}

	return -1
}
