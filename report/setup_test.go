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

//go:build (darwin && cgo) || (freebsd && cgo) || linux || windows
// +build darwin,cgo freebsd,cgo linux windows

package report

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestProcessName(t *testing.T) {
	tableTests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short ascii",
			input:    "processname",
			expected: "processname",
		},
		{
			name:     "long ascii",
			input:    "processnameiswaytoolong",
			expected: "processnameiswa",
		},
		{
			name:     "short utf8",
			input:    "🔥🔥",
			expected: "🔥🔥",
		},
		{
			name:     "long utf8",
			input:    "🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥",
			expected: "🔥🔥🔥",
		},
	}

	for _, tt := range tableTests {
		t.Run(tt.name, func(t *testing.T) {
			name := processName(tt.input)
			expected := tt.expected
			if isWindows() {
				// on Windows, no truncation is performed
				expected = tt.input
			}
			require.Truef(t, utf8.ValidString(name), "process name is invalid utf8: %q", name)
			require.Equal(t, expected, name)
		})
	}
}
