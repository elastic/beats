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

package integration

import (
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutputCounter(t *testing.T) {
	t.Run("Counter", func(t *testing.T) {
		cases := []struct {
			name      string
			strs      []string
			input     []string
			expOutput int64
		}{
			{
				name: "single line partially matches a single substring",
				strs: []string{
					"would match",
				},
				input: []string{
					"some log line that would match",
				},
				expOutput: 1,
			},
			{
				name: "single line does not match a single substring",
				strs: []string{
					"no line",
				},
				input: []string{
					"some log line that would not match",
				},
				expOutput: 0,
			},
			{
				name: "some lines match a single substring",
				strs: []string{
					"would match",
				},
				input: []string{
					"some log line that would match 1",
					"foo",
					"some log line that would match 2",
					"bar",
					"some log line that would match 3",
				},
				expOutput: 3,
			},
			{
				name: "lines match multiple substrings in order",
				strs: []string{
					"line foo",
					"line bar",
				},
				input: []string{
					"some log line that would not match 1",
					"line foo",
					"some log line that would not match 2",
					"line bar",
					"some log line that would not match 3",
					"line foo",
					"some log line that would not match 4",
					"line bar",
				},
				expOutput: 2,
			},
			{
				name: "lines match multiple substrings out of order",
				strs: []string{
					"line foo",
					"line bar",
				},
				input: []string{
					"1 line bar",
					"some log line that would not match 1",
					"2 line foo",
					"3 line foo",
					"some log line that would not match 2",
					"4 line bar",
					"some log line that would not match 3",
					"5 line foo",
					"some log line that would not match 4",
					"6 line bar",
				},
				expOutput: 2,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				out := &atomic.Int64{}
				w := NewCounter(out, tc.strs...)
				for _, s := range tc.input {
					w.Inspect(s)
				}

				require.Equal(t, tc.expOutput, out.Load())
			})
		}
	})

	t.Run("RegexpCounter", func(t *testing.T) {
		cases := []struct {
			name      string
			exprs     []*regexp.Regexp
			input     []string
			expOutput int64
		}{
			{
				name: "single line partially matches a single expression",
				exprs: []*regexp.Regexp{
					regexp.MustCompile("line(.*)match"),
				},
				input: []string{
					"some log line that would match",
				},
				expOutput: 1,
			},
			{
				name: "single line does not match a single expression",
				exprs: []*regexp.Regexp{
					regexp.MustCompile("no(.*)line"),
				},
				input: []string{
					"some log line that would not match",
				},
				expOutput: 0,
			},
			{
				name: "some lines match a single expression",
				exprs: []*regexp.Regexp{
					regexp.MustCompile("line(.*)match"),
				},
				input: []string{
					"some log line that would match 1",
					"foo",
					"some log line that would match 2",
					"bar",
					"some log line that would match 3",
				},
				expOutput: 3,
			},
			{
				name: "lines match multiple expressions in order",
				exprs: []*regexp.Regexp{
					regexp.MustCompile("line(.*)foo"),
					regexp.MustCompile("line(.*)bar"),
				},
				input: []string{
					"some log line that would not match 1",
					"line foo",
					"some log line that would not match 2",
					"line bar",
					"some log line that would not match 3",
					"line foo",
					"some log line that would not match 4",
					"line bar",
				},
				expOutput: 2,
			},
			{
				name: "lines match multiple expressions out of order",
				exprs: []*regexp.Regexp{
					regexp.MustCompile("line(.*)foo"),
					regexp.MustCompile("line(.*)bar"),
				},
				input: []string{
					"line bar",
					"some log line that would not match 1",
					"line foo",
					"line foo",
					"some log line that would not match 2",
					"line bar",
					"some log line that would not match 3",
					"line foo",
					"some log line that would not match 4",
					"line bar",
				},
				expOutput: 2,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				out := &atomic.Int64{}
				w := NewRegexpCounter(out, tc.exprs...)
				for _, s := range tc.input {
					w.Inspect(s)
				}

				require.Equal(t, tc.expOutput, out.Load())
			})
		}
	})
}
