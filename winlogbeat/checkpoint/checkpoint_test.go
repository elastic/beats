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

//go:build !integration

package checkpoint

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRead(t *testing.T) {
	writeFile := func(t *testing.T, content string) string {
		t.Helper()
		file := filepath.Join(t.TempDir(), ".winlogbeat.yml")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600), "failed to write fixture")
		return file
	}

	t.Run("missing file", func(t *testing.T) {
		got, err := Read(filepath.Join(t.TempDir(), "missing.yml"))
		assert.NoError(t, err, "missing file should not be an error")
		assert.Empty(t, got, "missing file should give an empty map")
	})

	t.Run("valid file", func(t *testing.T) {
		file := writeFile(t, `update_time: 2026-01-02T03:04:05Z
event_logs:
- name: Security
  record_number: 42
  timestamp: 2026-01-02T03:04:00Z
  bookmark: "<BookmarkList><Bookmark Channel='Security' RecordId='42'/></BookmarkList>"
- name: my-id
  record_number: 7
  timestamp: 2026-01-02T03:03:00Z
`)
		got, err := Read(file)
		require.NoError(t, err, "valid file should be read")
		want := map[string]EventLogState{
			"Security": {
				Name:         "Security",
				RecordNumber: 42,
				Timestamp:    time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC),
				Bookmark:     "<BookmarkList><Bookmark Channel='Security' RecordId='42'/></BookmarkList>",
			},
			"my-id": {
				Name:         "my-id",
				RecordNumber: 7,
				Timestamp:    time.Date(2026, 1, 2, 3, 3, 0, 0, time.UTC),
			},
		}
		require.Len(t, got, len(want), "unexpected number of states")
		for name, w := range want {
			g := got[name]
			assert.Equal(t, w.Name, g.Name, "name of %s", name)
			assert.Equal(t, w.RecordNumber, g.RecordNumber, "record number of %s", name)
			assert.True(t, w.Timestamp.Equal(g.Timestamp), "timestamp of %s: got %v want %v", name, g.Timestamp, w.Timestamp)
			assert.Equal(t, w.Bookmark, g.Bookmark, "bookmark of %s", name)
		}
	})

	t.Run("invalid file", func(t *testing.T) {
		file := writeFile(t, "event_logs: [")
		_, err := Read(file)
		require.Error(t, err, "invalid YAML should be an error")
		assert.Contains(t, err.Error(), file, "error should name the file")
	})

	t.Run("duplicate names", func(t *testing.T) {
		file := writeFile(t, `event_logs:
- name: Security
  record_number: 1
- name: Security
  record_number: 2
`)
		got, err := Read(file)
		require.NoError(t, err, "valid file should be read")
		assert.Len(t, got, 1, "duplicates should collapse to one entry")
		assert.Equal(t, uint64(2), got["Security"].RecordNumber, "last entry should win")
	})
}
