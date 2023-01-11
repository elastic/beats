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

package dissect

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const asciiLimit = 128

type trimmer interface {
	Trim(s string, start, end int) (int, int)
}

func newTrimmer(trimChars string, trimLeft, trimRight bool) (t trimmer, err error) {
	if t, err = newASCIITrimmer(trimChars, trimLeft, trimRight); err == errOnlyASCII {
		t, err = newUTF8Trimmer(trimChars, trimLeft, trimRight)
	}
	return t, err
}

type asciiTrimmer struct {
	chars       [asciiLimit]byte
	left, right bool
}

var errOnlyASCII = errors.New("only trimming of ASCII characters is supported")

func newASCIITrimmer(trimChars string, trimLeft, trimRight bool) (trimmer, error) {
	t := asciiTrimmer{
		left:  trimLeft,
		right: trimRight,
	}
	for _, chr := range []byte(trimChars) {
		if chr >= asciiLimit {
			return t, errOnlyASCII
		}
		t.chars[chr] = 1
	}
	return t, nil
}

func (t asciiTrimmer) Trim(s string, start, end int) (int, int) {
	if t.left {
		for ; start < end && s[start] < asciiLimit && t.chars[s[start]] != 0; start++ {
		}
	}
	if t.right {
		for ; start < end && s[end-1] < asciiLimit && t.chars[s[end-1]] != 0; end-- {
		}
	}
	return start, end
}

type utf8trimmer struct {
	fn          func(rune) bool
	left, right bool
}

func newUTF8Trimmer(trimChars string, trimLeft, trimRight bool) (trimmer, error) {
	return utf8trimmer{
		// Function that returns true when the rune is not in trimChars.
		fn: func(r rune) bool {
			return strings.IndexRune(trimChars, r) == -1
		},
		left:  trimLeft,
		right: trimRight,
	}, nil
}

func (t utf8trimmer) Trim(s string, start, end int) (int, int) {
	if t.left {
		// Find first character not in trimChars.
		pos := strings.IndexFunc(s[start:end], t.fn)
		if pos == -1 {
			return end, end
		}
		start += pos
	}
	if t.right {
		// Find last character not in trimChars.
		pos := strings.LastIndexFunc(s[start:end], t.fn)
		if pos == -1 {
			return start, start
		}
		// End must point to the following character, need to take into account
		// that the last character can be more than 1-byte wide.
		_, width := utf8.DecodeRuneInString(s[start+pos:])
		end = start + pos + width
	}
	return start, end
}
