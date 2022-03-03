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
	"testing"

	"github.com/stretchr/testify/assert"
)

type stripCase struct {
	s          string
	start, end int
}

func makeStrings(n, l int) []stripCase {
	cases := make([]stripCase, n)
	for idx := range cases {
		data := make([]byte, l)
		start := (idx / 2) % l
		end := l - ((idx+1)/2)%l
		if end < start {
			start, end = end, start
		}
		if start == end {
			start, end = l, l
		}
		for i := 0; i < start; i++ {
			data[i] = ' '
		}
		for i := start; i < end; i++ {
			data[i] = 'X'
		}
		for i := end; i < l; i++ {
			data[i] = ' '
		}
		cases[idx] = stripCase{string(data), start, end}
	}
	return cases
}

func benchStrip(b *testing.B, l int, t trimmer) {
	cases := makeStrings(b.N, l)
	b.ResetTimer()
	for idx, c := range cases {
		start, end := t.Trim(c.s, 0, len(c.s))
		if start != c.start || end != c.end {
			b.Logf("bad result idx=%d len=%d expected=(%d,%d) actual=(%d,%d)",
				idx, len(c.s), c.start, c.end, start, end)
			b.Fail()
		}
	}
}

func benchStripASCII(b *testing.B, l int) {
	trimmer, err := newASCIITrimmer(" ", true, true)
	if !assert.NoError(b, err) {
		b.Fail()
		return
	}
	benchStrip(b, l, trimmer)
}

func benchStripUTF8(b *testing.B, l int) {
	trimmer, err := newUTF8Trimmer(" ", true, true)
	if !assert.NoError(b, err) {
		b.Fail()
		return
	}
	benchStrip(b, l, trimmer)
}

func BenchmarkStripASCII_4(b *testing.B) {
	benchStripASCII(b, 4)
}

func BenchmarkStripASCII_8(b *testing.B) {
	benchStripASCII(b, 8)
}

func BenchmarkStripASCII_32(b *testing.B) {
	benchStripASCII(b, 32)
}

func BenchmarkStripASCII_128(b *testing.B) {
	benchStripASCII(b, 128)
}

func BenchmarkStripASCII_512(b *testing.B) {
	benchStripASCII(b, 512)
}

func BenchmarkStripUTF8_4(b *testing.B) {
	benchStripUTF8(b, 4)
}

func BenchmarkStripUTF8_8(b *testing.B) {
	benchStripUTF8(b, 8)
}

func BenchmarkStripUTF8_32(b *testing.B) {
	benchStripUTF8(b, 32)
}

func BenchmarkStripUTF8_128(b *testing.B) {
	benchStripUTF8(b, 128)
}

func BenchmarkStripUTF8_512(b *testing.B) {
	benchStripUTF8(b, 512)
}

func TestTrimmer(t *testing.T) {
	for _, test := range []struct {
		name, cutset    string
		left, right     bool
		input, expected string
	}{
		{
			name:     "single space right",
			cutset:   " ",
			right:    true,
			input:    " hello world! ",
			expected: " hello world!",
		},
		{
			name:     "noop right",
			cutset:   " ",
			right:    true,
			input:    "  hello world!",
			expected: "  hello world!",
		},
		{
			name:     "single space left",
			cutset:   " ",
			left:     true,
			input:    " hello world! ",
			expected: "hello world! ",
		},
		{
			name:     "noop left",
			cutset:   " ",
			left:     true,
			input:    "hello world!  ",
			expected: "hello world!  ",
		},
		{
			name:     "trim both",
			cutset:   " ",
			left:     true,
			right:    true,
			input:    "  hello world!  ",
			expected: "hello world!",
		},
		{
			name:     "non-space",
			cutset:   "h",
			left:     true,
			right:    true,
			input:    "hello world!",
			expected: "ello world!",
		},
		{
			name:     "multiple chars",
			cutset:   " \t_-",
			left:     true,
			right:    true,
			input:    "\t\t___here - -",
			expected: "here",
		},
		{
			name:     "empty string",
			cutset:   " \t_-",
			left:     true,
			right:    true,
			input:    "",
			expected: "",
		},
		{
			name:     "trim all",
			cutset:   " \t_-",
			left:     true,
			right:    true,
			input:    " \t__-",
			expected: "",
		},
		{
			name:     "trim UTF-8",
			cutset:   "߹༄𑁍",
			left:     true,
			right:    true,
			input:    "༄𑅀߹꧁߹𑁍",
			expected: "𑅀߹꧁",
		},
		{
			name:     "trim ASCII cutset in UTF-8 input",
			cutset:   " \t\rÿ",
			left:     true,
			right:    true,
			input:    "\t\t༄𑅀߹꧁߹𑁍 ÿ",
			expected: "༄𑅀߹꧁߹𑁍",
		},
		{ // demonstrates that unicode \u is converted to char in golang strings
			name:     "trim ASCII TILDE",
			cutset:   " ",
			left:     true,
			right:    true,
			input:    "  hello world! \u007e ",
			expected: "hello world! ~",
		},
		{
			name:     "trim ASCII DELETE",
			cutset:   " ",
			left:     true,
			right:    true,
			input:    "  hello world! \u007f ",
			expected: "hello world! \u007f",
		},
		{
			name:     "trim UTF-8 CONTROL",
			cutset:   " ",
			left:     true,
			right:    true,
			input:    "  hello world! \u0080 ",
			expected: "hello world! \u0080",
		},
		{
			name:     "trim ASCII DELETE cutset in UTF-8 input",
			cutset:   " \u007f",
			left:     true,
			right:    true,
			input:    "  hello world! \u0080 \u007f",
			expected: "hello world! \u0080",
		},
		{
			name:     "trim UTF-8 CONTROL cutset in UTF-8 input",
			cutset:   " \u0080",
			left:     true,
			right:    true,
			input:    "  hello world! \u007f \u0080",
			expected: "hello world! \u007f",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			trimmer, err := newTrimmer(test.cutset, test.left, test.right)
			if !assert.NoError(t, err) {
				return
			}
			start, end := trimmer.Trim(test.input, 0, len(test.input))
			output := test.input[start:end]
			assert.Equal(t, test.expected, output)
		})
	}
}
