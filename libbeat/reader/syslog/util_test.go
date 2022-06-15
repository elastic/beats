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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestStringToInt(t *testing.T) {
	cases := map[string]struct {
		in   string
		want int
	}{
		"valid_1024": {
			in:   "1024",
			want: 1024,
		},
		"valid_-1024": {
			in:   "-1024",
			want: -1024,
		},
		"valid_0": {
			in:   "0",
			want: 0,
		},
		"invalid_-": {
			in:   "-",
			want: 0,
		},
		"invalid_+": {
			in:   "+",
			want: 0,
		},
		"invalid_empty": {
			in:   "",
			want: 0,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := stringToInt(tc.in)

			assert.Equal(t, tc.want, got)
		})
	}
}

var removeBytesCases = map[string]struct {
	in        string
	positions []int
	offset    int
	want      string
}{
	"basic-1": {
		in:        "abcdefghijklmno",
		positions: []int{3, 5, 6, 10, 12},
		want:      "abcehijlno",
	},
	"basic-2": {
		in:        "abcdefghijklmno",
		positions: []int{0, 5, 6, 10, 12},
		want:      "bcdehijlno",
	},
	"basic-3": {
		in:        "abcdefghijklmno",
		positions: []int{0, 1, 2, 3, 4},
		want:      "fghijklmno",
	},
	"basic-4": {
		in:        "\\ab\\cd\\ef\\ghijklmno",
		positions: []int{0, 1, 2, 9},
		want:      "\\cd\\efghijklmno",
	},
	"offset-1": {
		in:        "abcdefghijklmno",
		positions: []int{5, 8, 9, 12},
		offset:    5,
		want:      "bcfgijklmno",
	},
	"no-positions": {
		in:        "abcdefghijklmno",
		positions: nil,
		want:      "abcdefghijklmno",
	},
	"negative-offset": {
		in:        "abcdefghijklmno",
		positions: []int{5, 8, 9, 12},
		offset:    -1,
		want:      "abcdefghijklmno",
	},
	"too-many-positions": {
		in:        "abc",
		positions: []int{5, 8, 9, 12},
		want:      "abc",
	},
	"offset-position-negative": {
		in:        "abc",
		positions: []int{3},
		offset:    5,
		want:      "abc",
	},
	"offset-position-out-of-bounds": {
		in:        "abc",
		positions: []int{8},
		offset:    5,
		want:      "abc",
	},
}

func TestRemoveBytes(t *testing.T) {
	for name, tc := range removeBytesCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := removeBytes(tc.in, tc.positions, tc.offset)

			assert.Equal(t, tc.want, got)
		})
	}
}

func BenchmarkRemoveBytes(b *testing.B) {
	for name, bc := range removeBytesCases {
		bc := bc
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = removeBytes(bc.in, bc.positions, bc.offset)
			}
		})
	}
}

func TestMapIndexToString(t *testing.T) {
	items := []string{
		"A",
		"B",
		"C",
	}
	tests := map[string]struct {
		in     int
		want   string
		wantOK bool
	}{
		"valid-index-bottom": {
			in:     0,
			want:   items[0],
			wantOK: true,
		},
		"valid-index-top": {
			in:     len(items) - 1,
			want:   items[len(items)-1],
			wantOK: true,
		},
		"invalid-index-high": {
			in:     len(items),
			wantOK: false,
		},
		"invalid-index-low": {
			in:     -1,
			wantOK: false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, ok := mapIndexToString(tc.in, items)

			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantOK, ok)
		})
	}
}

func TestAppendStringField(t *testing.T) {
	tests := map[string]struct {
		inMap   mapstr.M
		inField string
		inValue string
		want    mapstr.M
	}{
		"nil": {
			inMap:   mapstr.M{},
			inField: "error",
			inValue: "foo",
			want: mapstr.M{
				"error": "foo",
			},
		},
		"string": {
			inMap: mapstr.M{
				"error": "foo",
			},
			inField: "error",
			inValue: "bar",
			want: mapstr.M{
				"error": []string{"foo", "bar"},
			},
		},
		"string-slice": {
			inMap: mapstr.M{
				"error": []string{"foo", "bar"},
			},
			inField: "error",
			inValue: "some value",
			want: mapstr.M{
				"error": []string{"foo", "bar", "some value"},
			},
		},
		"interface-slice": {
			inMap: mapstr.M{
				"error": []interface{}{"foo", "bar"},
			},
			inField: "error",
			inValue: "some value",
			want: mapstr.M{
				"error": []interface{}{"foo", "bar", "some value"},
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			appendStringField(tc.inMap, tc.inField, tc.inValue)

			assert.Equal(t, tc.want, tc.inMap)
		})
	}
}

func TestJoinStr(t *testing.T) {
	tests := map[string]struct {
		inA   string
		inB   string
		inSep string
		want  string
	}{
		"both-empty": {
			inA:   "",
			inB:   "",
			inSep: " ",
			want:  "",
		},
		"only-a": {
			inA:   "alpha",
			inB:   "",
			inSep: " ",
			want:  "alpha",
		},
		"only-b": {
			inA:   "",
			inB:   "beta",
			inSep: " ",
			want:  "beta",
		},
		"both": {
			inA:   "alpha",
			inB:   "beta",
			inSep: " ",
			want:  "alpha beta",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := joinStr(tc.inA, tc.inB, tc.inSep)

			assert.Equal(t, tc.want, got)
		})
	}
}
