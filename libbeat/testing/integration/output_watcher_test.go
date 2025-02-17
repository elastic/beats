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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutputWatcher(t *testing.T) {
	t.Run("StringWatcher", func(t *testing.T) {
		cases := []struct {
			name           string
			str            string
			expectObserved bool
			input          string
		}{
			{
				name:           "line has substring",
				str:            "log line",
				expectObserved: true,
				input:          "some log line that would match",
			},
			{
				name:           "line does not have substring",
				str:            "no line",
				expectObserved: false,
				input:          "some log line that would match",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				w := NewStringWatcher(tc.str)
				w.Inspect(tc.input)
				require.Equal(t, tc.expectObserved, w.Observed(), "Observed() does not match")
			})
		}
	})

	t.Run("RegexpWatcher", func(t *testing.T) {
		cases := []struct {
			name           string
			expr           *regexp.Regexp
			expectObserved bool
			input          string
		}{
			{
				name:           "line partially matches",
				expr:           regexp.MustCompile("line(.*)match"),
				expectObserved: true,
				input:          "some log line that would match",
			},
			{
				name:           "line does not match",
				expr:           regexp.MustCompile("no(.*)line"),
				expectObserved: false,
				input:          "some log line that would match",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				w := NewRegexpWatcher(tc.expr)
				w.Inspect(tc.input)
				require.Equal(t, tc.expectObserved, w.Observed(), "Observed() does not match")
			})
		}
	})

	t.Run("InOrderWatcher", func(t *testing.T) {
		cases := []struct {
			name           string
			strs           []string
			expectObserved bool
			input          []string
		}{
			{
				name:           "lines match in order",
				strs:           []string{"first match", "second match"},
				expectObserved: true,
				input: []string{
					"not important line",
					"this would trigger the first match",
					"not important line",
					"not important line",
					"this would trigger the second match",
					"not important line",
				},
			},
			{
				name:           "lines don't match in order",
				strs:           []string{"first match", "second match"},
				expectObserved: false,
				input: []string{
					"not important line",
					"this would trigger the second match",
					"not important line",
					"not important line",
					"this would trigger the first match",
					"not important line",
				},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				watchers := make([]OutputWatcher, 0, len(tc.strs))
				for _, s := range tc.strs {
					watchers = append(watchers, NewStringWatcher(s))
				}
				w := NewInOrderWatcher(watchers)

				for _, l := range tc.input {
					w.Inspect(l)
				}

				require.Equal(t, tc.expectObserved, w.Observed(), "Observed() does not match")
			})
		}
	})

	t.Run("OverallWatcher", func(t *testing.T) {
		cases := []struct {
			name           string
			strs           []string
			expectObserved bool
			input          []string
		}{
			{
				name:           "lines match in order",
				strs:           []string{"first match", "second match"},
				expectObserved: true,
				input: []string{
					"not important line",
					"this would trigger the first match",
					"not important line",
					"not important line",
					"this would trigger the second match",
					"not important line",
				},
			},
			{
				name:           "lines don't match in order",
				strs:           []string{"first match", "no second match"},
				expectObserved: false,
				input: []string{
					"not important line",
					"this would trigger the first match",
					"not important line",
					"not important line",
					"this would trigger the second match",
					"not important line",
				},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				watchers := make([]OutputWatcher, 0, len(tc.strs))
				for _, s := range tc.strs {
					watchers = append(watchers, NewStringWatcher(s))
				}
				w := NewOverallWatcher(watchers)

				for _, l := range tc.input {
					w.Inspect(l)
				}

				require.Equal(t, tc.expectObserved, w.Observed(), "Observed() does not match")
			})
		}
	})
}
