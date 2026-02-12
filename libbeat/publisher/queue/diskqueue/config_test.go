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

package diskqueue

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/paths"
)

func TestDirectoryPath(t *testing.T) {
	tests := map[string]struct {
		settings Settings
		expected string
	}{
		"explicit path takes precedence": {
			settings: Settings{
				Path: "/custom/queue/path",
				Paths: &paths.Path{
					Data: "/beat/data",
				},
			},
			expected: "/custom/queue/path",
		},
		"per-beat paths used when Path is empty": {
			settings: Settings{
				Paths: &paths.Path{
					Data: "/beat/data",
				},
			},
			expected: filepath.Join("/beat/data", "diskqueue"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := test.settings.directoryPath()
			assert.Equal(t, test.expected, result)
		})
	}
}
