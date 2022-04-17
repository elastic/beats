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

	"github.com/menderesk/beats/v7/libbeat/common"
)

func TestStringToInt(t *testing.T) {
	cases := map[string]struct {
		In   string
		Want int
	}{
		"valid_1024": {
			In:   "1024",
			Want: 1024,
		},
		"valid_-1024": {
			In:   "-1024",
			Want: -1024,
		},
		"valid_0": {
			In:   "0",
			Want: 0,
		},
		"invalid_-": {
			In:   "-",
			Want: 0,
		},
		"invalid_+": {
			In:   "+",
			Want: 0,
		},
		"invalid_empty": {
			In:   "",
			Want: 0,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := stringToInt(tc.In)

			assert.Equal(t, tc.Want, got)
		})
	}
}

var removeBytesCases = map[string]struct {
	In        string
	Positions []int
	Offset    int
	Want      string
}{
	"basic-1": {
		In:        "abcdefghijklmno",
		Positions: []int{3, 5, 6, 10, 12},
		Want:      "abcehijlno",
	},
	"basic-2": {
		In:        "abcdefghijklmno",
		Positions: []int{0, 5, 6, 10, 12},
		Want:      "bcdehijlno",
	},
	"basic-3": {
		In:        "abcdefghijklmno",
		Positions: []int{0, 1, 2, 3, 4},
		Want:      "fghijklmno",
	},
	"basic-4": {
		In:        "\\ab\\cd\\ef\\ghijklmno",
		Positions: []int{0, 1, 2, 9},
		Want:      "\\cd\\efghijklmno",
	},
	"offset-1": {
		In:        "abcdefghijklmno",
		Positions: []int{5, 8, 9, 12},
		Offset:    5,
		Want:      "bcfgijklmno",
	},
	"no-positions": {
		In:        "abcdefghijklmno",
		Positions: nil,
		Want:      "abcdefghijklmno",
	},
	"negative-offset": {
		In:        "abcdefghijklmno",
		Positions: []int{5, 8, 9, 12},
		Offset:    -1,
		Want:      "abcdefghijklmno",
	},
	"too-many-positions": {
		In:        "abc",
		Positions: []int{5, 8, 9, 12},
		Want:      "abc",
	},
	"offset-position-negative": {
		In:        "abc",
		Positions: []int{3},
		Offset:    5,
		Want:      "abc",
	},
	"offset-position-out-of-bounds": {
		In:        "abc",
		Positions: []int{8},
		Offset:    5,
		Want:      "abc",
	},
}

func TestRemoveBytes(t *testing.T) {
	for name, tc := range removeBytesCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := removeBytes(tc.In, tc.Positions, tc.Offset)

			assert.Equal(t, tc.Want, got)
		})
	}
}

func BenchmarkRemoveBytes(b *testing.B) {
	for name, bc := range removeBytesCases {
		bc := bc
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = removeBytes(bc.In, bc.Positions, bc.Offset)
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
		In     int
		Want   string
		WantOK bool
	}{
		"valid-index-bottom": {
			In:     0,
			Want:   items[0],
			WantOK: true,
		},
		"valid-index-top": {
			In:     len(items) - 1,
			Want:   items[len(items)-1],
			WantOK: true,
		},
		"invalid-index-high": {
			In:     len(items),
			WantOK: false,
		},
		"invalid-index-low": {
			In:     -1,
			WantOK: false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, ok := mapIndexToString(tc.In, items)

			assert.Equal(t, tc.Want, got)
			assert.Equal(t, tc.WantOK, ok)
		})
	}
}

func TestAppendStringField(t *testing.T) {
	tests := map[string]struct {
		InMap   common.MapStr
		InField string
		InValue string
		Want    common.MapStr
	}{
		"nil": {
			InMap:   common.MapStr{},
			InField: "error",
			InValue: "foo",
			Want: common.MapStr{
				"error": "foo",
			},
		},
		"string": {
			InMap: common.MapStr{
				"error": "foo",
			},
			InField: "error",
			InValue: "bar",
			Want: common.MapStr{
				"error": []string{"foo", "bar"},
			},
		},
		"string-slice": {
			InMap: common.MapStr{
				"error": []string{"foo", "bar"},
			},
			InField: "error",
			InValue: "some value",
			Want: common.MapStr{
				"error": []string{"foo", "bar", "some value"},
			},
		},
		"interface-slice": {
			InMap: common.MapStr{
				"error": []interface{}{"foo", "bar"},
			},
			InField: "error",
			InValue: "some value",
			Want: common.MapStr{
				"error": []interface{}{"foo", "bar", "some value"},
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			appendStringField(tc.InMap, tc.InField, tc.InValue)

			assert.Equal(t, tc.Want, tc.InMap)
		})
	}
}
