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

package filestream

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// hookedSink lets a test mutate the filesystem around the processing of each
// listed path, reproducing a scan that raced with the change.
type hookedSink struct {
	*scanState
	around func(path string, process func())
}

func (h hookedSink) process(path string, order int) {
	h.around(path, func() { h.scanState.process(path, order) })
}

// scanDuringChange runs a real walk whose every matched path is handed to
// around, which decides when to process it relative to its own mutation.
func scanDuringChange(s *fileScanner, around func(path string, process func())) loginp.ScanResults {
	sink := hookedSink{scanState: s.newScanState(loginp.FileScanOptions{CurrentTime: time.Now()}), around: around}
	for _, g := range s.walkGroups {
		s.walk(g, sink)
	}
	return sink.results()
}

// watchScan feeds one scan result through the watcher's event logic.
func watchScan(w *fileWatcher, result loginp.ScanResults) []loginp.FSEvent {
	w.scanner = &queuedScanner{scans: []scanResult{{
		files:        result.Files,
		unobservable: result.Unobservable,
		vanished:     result.Vanished,
	}}}
	metrics := newTestMetrics()
	defer metrics.Cleanup()
	w.watch(context.Background(), metrics, 0, time.Time{})
	return drainPendingFSEvents(w.events)
}

// trackedWatcher returns a watcher that has already seen want created by s.
func trackedWatcher(t *testing.T, s *fileScanner, want ...string) *fileWatcher {
	t.Helper()
	w := newStubWatcher(s)
	expected := make([]loginp.FSEvent, len(want))
	for i, path := range want {
		expected[i] = loginp.FSEvent{Op: loginp.OpCreate, NewPath: path}
	}
	requireEventSignatures(t, watchScan(w, s.GetFiles(loginp.FileScanOptions{})), expected)
	return w
}

func TestFileWatcherRenameDuringScan(t *testing.T) {
	// Without a cache the only way a scan can race is mid-walk; with one the
	// listing is already stale by the time GetFiles runs.
	modes := []struct {
		name         string
		cached       bool
		refreshOther bool
	}{
		{name: "uncached"},
		{name: "cached", cached: true},
		{name: "other_scanner_refresh", cached: true, refreshOther: true},
	}
	identities := []struct {
		name            string
		fingerprint     fingerprintConfig
		contentLen      int
		growAfterRename bool
	}{
		{name: "native", contentLen: 32},
		{name: "static", fingerprint: fingerprintConfig{Enabled: true, Length: 64}, contentLen: 64},
		{name: "growing", fingerprint: fingerprintConfig{Enabled: true, Length: 64, Growing: true}, contentLen: 32},
		{name: "threshold_crossing", fingerprint: fingerprintConfig{Enabled: true, Length: 64, Growing: true}, contentLen: 32, growAfterRename: true},
	}

	for _, mode := range modes {
		for _, identity := range identities {
			t.Run(mode.name+"/"+identity.name, func(t *testing.T) {
				dir := t.TempDir()
				oldPath := filepath.Join(dir, "app.log")
				newPath := oldPath + ".1"
				content := strings.Repeat("a", identity.contentLen)
				require.NoError(t, os.WriteFile(oldPath, []byte(content), 0o600), "create original file")

				var dc *dirCache
				if mode.cached {
					dc = newDirCache()
				}
				paths := []string{filepath.Join(dir, "*.log*")}
				cfg := fileScannerConfig{Fingerprint: identity.fingerprint}
				s, err := newFileScannerWithCache(logptest.NewTestingLogger(t, ""), paths, cfg, CompressionNone, dc, time.Hour)
				require.NoError(t, err, "create scanner")
				w := trackedWatcher(t, s, oldPath)
				w.growingFingerprint = identity.fingerprint.Growing

				if mode.refreshOther {
					// A listing fetched by a different scanner can still predate the
					// rename, even if it is newer than this watcher's last scan.
					dc.entry(dir).fetched.Store(0)
					other, err := newFileScannerWithCache(logptest.NewTestingLogger(t, ""), paths, cfg, CompressionNone, dc, time.Hour)
					require.NoError(t, err, "create another scanner sharing the cache")
					assert.Contains(t, other.GetFiles(loginp.FileScanOptions{}).Files, oldPath, "other scanner must refresh before the rename")
				}

				renamed := false
				rename := func() {
					renamed = true
					require.NoError(t, os.Rename(oldPath, newPath), "rename after listing and before stat")
					if identity.growAfterRename {
						require.NoError(t, os.WriteFile(newPath, []byte(content+content), 0o600), "grow renamed file past the fingerprint threshold")
					}
				}
				var inconsistent loginp.ScanResults
				if mode.cached {
					rename()
					inconsistent = s.GetFiles(loginp.FileScanOptions{})
				} else {
					inconsistent = scanDuringChange(s, func(path string, process func()) {
						if path == oldPath && !renamed {
							rename()
						}
						process()
					})
				}
				require.True(t, renamed, "test must rename the listed file")
				assert.Empty(t, inconsistent.Files, "neither old nor new path can be read from the stale listing")
				assert.Equal(t, []string{oldPath}, inconsistent.Vanished, "listed-but-missing file must be preserved")
				assert.Empty(t, inconsistent.Unobservable, "a rename is not an observation failure")
				assert.Zero(t, inconsistent.Metrics.ScanErrors, "a rename must not be counted as a scan error")
				assert.Empty(t, watchScan(w, inconsistent), "an inconsistent scan must not delete the old state")
				assert.Contains(t, w.prev, oldPath, "previous descriptor must survive until rename detection")

				if dc != nil {
					// A repeat of the stale listing must not cause deletion either.
					assert.Empty(t, watchScan(w, s.GetFiles(loginp.FileScanOptions{})), "keep state while the cache remains stale")
					dc.entry(dir).fetched.Store(0)
				}
				requireEventSignatures(t, watchScan(w, s.GetFiles(loginp.FileScanOptions{})), []loginp.FSEvent{
					{Op: loginp.OpRename, OldPath: oldPath, NewPath: newPath},
				})
				assert.NotContains(t, w.prev, oldPath, "renamed descriptor must not remain under its old path")
				assert.Contains(t, w.prev, newPath, "new path must now be tracked")
				assert.Empty(t, watchScan(w, s.GetFiles(loginp.FileScanOptions{})), "rename must not be emitted twice")
			})
		}
	}
}

func TestFileWatcherDeleteDuringScan(t *testing.T) {
	for _, mode := range []struct {
		name   string
		cached bool
	}{{name: "uncached"}, {name: "cached", cached: true}} {
		t.Run(mode.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "app.log")
			require.NoError(t, os.WriteFile(path, []byte("hello"), 0o600), "create file")
			var dc *dirCache
			if mode.cached {
				dc = newDirCache()
			}
			s, err := newFileScannerWithCache(logptest.NewTestingLogger(t, ""), []string{filepath.Join(dir, "*.log")}, fileScannerConfig{}, CompressionNone, dc, time.Hour)
			require.NoError(t, err, "create scanner")
			w := trackedWatcher(t, s, path)

			inconsistent := scanDuringChange(s, func(listed string, process func()) {
				require.Equal(t, path, listed, "unexpected listed path")
				require.NoError(t, os.Remove(path), "delete after listing and before stat")
				process()
			})
			assert.Zero(t, inconsistent.Metrics.ScanErrors, "an ordinary delete must not be counted as a scan error")
			assert.Empty(t, watchScan(w, inconsistent), "deletion needs a consistent scan to be confirmed")
			if dc != nil {
				assert.Empty(t, watchScan(w, s.GetFiles(loginp.FileScanOptions{})), "deletion must remain deferred while the listing is stale")
				dc.entry(dir).fetched.Store(0)
			}
			requireEventSignatures(t, watchScan(w, s.GetFiles(loginp.FileScanOptions{})), []loginp.FSEvent{{Op: loginp.OpDelete, OldPath: path}})
			assert.Empty(t, w.prev, "confirmed deletion must release previous state")
		})
	}
}

func TestFileWatcherRenameBeforeFingerprintOpen(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "app.log")
	newPath := oldPath + ".1"
	require.NoError(t, os.WriteFile(oldPath, []byte(strings.Repeat("a", 64)), 0o600), "create file")
	cfg := fileScannerConfig{Fingerprint: fingerprintConfig{Enabled: true, Length: 64}}
	s, err := newFileScanner(logptest.NewTestingLogger(t, ""), []string{filepath.Join(dir, "*.log*")}, cfg, CompressionNone)
	require.NoError(t, err, "create scanner")
	w := trackedWatcher(t, s, oldPath)

	it, err := s.getIngestTarget(oldPath)
	require.NoError(t, err, "stat must succeed before the rename")
	require.NoError(t, os.Rename(oldPath, newPath), "rename before opening for fingerprinting")
	_, err = s.toFileDescriptor(&it)
	require.ErrorIs(t, err, os.ErrNotExist, "fingerprint open must fail for the old path")
	st := s.newScanState(loginp.FileScanOptions{})
	st.recordPathError(oldPath, err)
	inconsistent := st.results()
	assert.Equal(t, []string{oldPath}, inconsistent.Vanished, "the path must be held for the next scan")
	assert.Zero(t, inconsistent.Metrics.ScanErrors, "a vanished path must not be counted as a scan error")
	assert.Empty(t, watchScan(w, inconsistent), "failed fingerprint open must not delete previous state")
	requireEventSignatures(t, watchScan(w, s.GetFiles(loginp.FileScanOptions{})), []loginp.FSEvent{{Op: loginp.OpRename, OldPath: oldPath, NewPath: newPath}})
}

func TestFileWatcherDirectoryRenameDuringScan(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, "old")
	newDir := filepath.Join(root, "new")
	oldPath := filepath.Join(oldDir, "app.log")
	newPath := filepath.Join(newDir, "app.log")
	require.NoError(t, os.Mkdir(oldDir, 0o700), "create directory")
	require.NoError(t, os.WriteFile(oldPath, []byte("hello"), 0o600), "create file")
	s, err := newFileScanner(logptest.NewTestingLogger(t, ""), []string{filepath.Join(root, "*"), filepath.Join(root, "*", "*.log")}, fileScannerConfig{}, CompressionNone)
	require.NoError(t, err, "create scanner")
	w := trackedWatcher(t, s, oldPath)

	inconsistent := scanDuringChange(s, func(path string, process func()) {
		// The directory's own stat succeeds; only the descent into it fails.
		process()
		if path == oldDir {
			require.NoError(t, os.Rename(oldDir, newDir), "rename listed directory before descent")
		}
	})
	assert.Contains(t, inconsistent.Vanished, oldDir, "missing listed directory must preserve its subtree")
	assert.Zero(t, inconsistent.Metrics.ScanErrors, "a renamed directory must not be counted as a scan error")
	assert.Empty(t, watchScan(w, inconsistent), "inconsistent descent must not delete tracked files")
	requireEventSignatures(t, watchScan(w, s.GetFiles(loginp.FileScanOptions{})), []loginp.FSEvent{{Op: loginp.OpRename, OldPath: oldPath, NewPath: newPath}})
}

func TestFileWatcherDanglingSymlinkIsRemoved(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires elevated privileges on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "target")
	link := filepath.Join(dir, "app.log")
	require.NoError(t, os.WriteFile(target, []byte("hello"), 0o600), "create target")
	require.NoError(t, os.Symlink(target, link), "create symlink")
	s, err := newFileScanner(logptest.NewTestingLogger(t, ""), []string{filepath.Join(dir, "*.log")}, fileScannerConfig{Symlinks: true}, CompressionNone)
	require.NoError(t, err, "create scanner")
	w := trackedWatcher(t, s, link)

	require.NoError(t, os.Remove(target), "remove target while keeping the symlink")
	result := s.GetFiles(loginp.FileScanOptions{})
	assert.Empty(t, result.Vanished, "a permanently dangling symlink must not defer deletion forever")
	assert.Empty(t, result.Unobservable, "a dangling symlink target is not an observation failure")
	assert.Zero(t, result.Metrics.ScanErrors, "a dangling symlink must not be counted as a scan error")
	requireEventSignatures(t, watchScan(w, result), []loginp.FSEvent{{Op: loginp.OpDelete, OldPath: link}})
	assert.Empty(t, w.prev, "dangling symlink state must be released")
}

func TestFileScannerMissingLiteralIsRemoved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o600), "create file")
	s, err := newFileScanner(logptest.NewTestingLogger(t, ""), []string{path}, fileScannerConfig{}, CompressionNone)
	require.NoError(t, err, "create scanner")
	w := trackedWatcher(t, s, path)

	require.NoError(t, os.Remove(path), "remove literal before discovery")
	result := s.GetFiles(loginp.FileScanOptions{})
	assert.Empty(t, result.Vanished, "a literal absent at initial discovery must not be deferred")
	assert.Empty(t, result.Unobservable, "an absent literal is not an observation failure")
	assert.Zero(t, result.Metrics.ScanErrors, "an absent literal is not an inconsistent observation")
	requireEventSignatures(t, watchScan(w, result), []loginp.FSEvent{{Op: loginp.OpDelete, OldPath: path}})
}
