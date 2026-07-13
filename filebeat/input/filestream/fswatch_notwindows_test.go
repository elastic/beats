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

//go:build !windows

package filestream

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// TestFileScannerReportsUnobservablePaths is the scanner half of the fd-exhaustion
// fix: a directory or file the scan cannot read/stat because of a resource or
// permission error (not because it is gone) must be reported as an unobservable
// prefix, and counted in ScanErrors, so the watcher can postpone deleting the
// files under it.
func TestFileScannerReportsUnobservablePaths(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-based test cannot work as root")
	}
	logger := logptest.NewTestingLogger(t, "")

	t.Run("unreadable directory is reported, readable sibling still scanned", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "ok"), 0o770))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "ok", "a.log"), []byte("hello\n"), 0o640))
		blocked := filepath.Join(dir, "blocked")
		require.NoError(t, os.MkdirAll(blocked, 0o770))
		require.NoError(t, os.WriteFile(filepath.Join(blocked, "b.log"), []byte("hello\n"), 0o640))
		require.NoError(t, os.Chmod(blocked, 0))
		t.Cleanup(func() { _ = os.Chmod(blocked, 0o770) }) // let TempDir cleanup remove it

		cfg := fileScannerConfig{RecursiveGlob: true, Fingerprint: fingerprintConfig{Enabled: false}}
		s, err := newFileScanner(logger, []string{filepath.Join(dir, "**", "*.log")}, cfg, CompressionNone)
		require.NoError(t, err)

		files, metrics, unobservable := s.GetFiles(loginp.FileScanOptions{})
		assert.Contains(t, files, filepath.Join(dir, "ok", "a.log"), "readable sibling must still be scanned")
		assert.NotContains(t, files, filepath.Join(blocked, "b.log"), "file under an unreadable dir cannot be observed")
		assert.Contains(t, unobservable, blocked, "unreadable dir must be reported as an unobservable prefix")
		assert.Positive(t, metrics.ScanErrors, "scan_errors must count the unobservable path")
	})

	t.Run("literal path under an unreadable directory is reported", func(t *testing.T) {
		dir := t.TempDir()
		blocked := filepath.Join(dir, "blocked")
		require.NoError(t, os.MkdirAll(blocked, 0o770))
		lit := filepath.Join(blocked, "b.log")
		require.NoError(t, os.WriteFile(lit, []byte("hello\n"), 0o640))
		require.NoError(t, os.Chmod(blocked, 0))
		t.Cleanup(func() { _ = os.Chmod(blocked, 0o770) })

		cfg := fileScannerConfig{Fingerprint: fingerprintConfig{Enabled: false}}
		s, err := newFileScanner(logger, []string{lit}, cfg, CompressionNone) // no glob meta: a literal
		require.NoError(t, err)

		files, metrics, unobservable := s.GetFiles(loginp.FileScanOptions{})
		assert.Empty(t, files, "the literal cannot be observed")
		assert.Contains(t, unobservable, lit, "literal we could not lstat must be reported")
		assert.Positive(t, metrics.ScanErrors)
	})
}
