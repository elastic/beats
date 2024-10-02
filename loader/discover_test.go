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

package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const agentConfigFile = "elastic-agent.yml"

func TestDiscover(t *testing.T) {
	t.Run("support wildcards patterns", withFiles([]string{"hello", "helllooo"}, func(
		dst string,
		t *testing.T,
	) {
		r, err := DiscoverFiles(filepath.Join(dst, "hel*"))
		require.NoError(t, err)
		assert.Equal(t, 2, len(r))
	}))

	t.Run("support direct file", withFiles([]string{"hello", "helllooo"}, func(
		dst string,
		t *testing.T,
	) {
		r, err := DiscoverFiles(filepath.Join(dst, "hello"))
		require.NoError(t, err)
		assert.Equal(t, 1, len(r))
	}))

	t.Run("support direct file and pattern", withFiles([]string{"hello", "helllooo", agentConfigFile}, func(
		dst string,
		t *testing.T,
	) {
		r, err := DiscoverFiles(
			filepath.Join(dst, "hel*"),
			filepath.Join(dst, agentConfigFile),
		)
		require.NoError(t, err)
		assert.Equal(t, 3, len(r))
	}))

	t.Run("support direct file and pattern", withFiles([]string{"hello", "helllooo", agentConfigFile}, func(
		dst string,
		t *testing.T,
	) {
		r, err := DiscoverFiles(filepath.Join(dst, "donotmatch.yml"))
		require.NoError(t, err)
		assert.Equal(t, 0, len(r))
	}))
}

func withFiles(files []string, fn func(dst string, t *testing.T)) func(t *testing.T) {
	return func(t *testing.T) {
		tmp := t.TempDir()

		for _, file := range files {
			path := filepath.Join(tmp, file)
			empty, _ := os.Create(path)
			empty.Close()
		}

		fn(tmp, t)
	}
}
