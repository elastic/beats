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

//nolint:errcheck // It's a test file
package filestream

import (
	"context"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	testingfs "github.com/elastic/elastic-agent-libs/testing/fs"
	"github.com/elastic/go-concert/unison"
)

func TestProspector_InitCleanIfRemoved(t *testing.T) {
	testCases := map[string]struct {
		entries             map[string]loginp.Value
		filesOnDisk         map[string]loginp.FileDescriptor
		cleanRemoved        bool
		expectedCleanedKeys []string
	}{
		"prospector init with clean_removed enabled with no entries": {
			entries:             nil,
			filesOnDisk:         nil,
			cleanRemoved:        true,
			expectedCleanedKeys: []string{},
		},
		"prospector init with clean_removed disabled with entries": {
			entries: map[string]loginp.Value{
				"key1": &mockUnpackValue{
					key: "key1",
					fileMeta: fileMeta{
						Source:         "/no/such/path",
						IdentifierName: "path",
					},
				},
			},
			filesOnDisk:         nil,
			cleanRemoved:        false,
			expectedCleanedKeys: []string{},
		},
		"prospector init with clean_removed enabled with entries": {
			entries: map[string]loginp.Value{
				"key1": &mockUnpackValue{
					key: "key1",
					fileMeta: fileMeta{
						Source:         "/no/such/path",
						IdentifierName: "path",
					},
				},
			},
			filesOnDisk:         nil,
			cleanRemoved:        true,
			expectedCleanedKeys: []string{"key1"},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			testStore := newMockStoreUpdater(testCase.entries)
			p := fileProspector{
				logger:       logp.NewNopLogger(),
				identifier:   mustPathIdentifier(false),
				cleanRemoved: testCase.cleanRemoved,
				filewatcher:  newMockFileWatcherWithFiles(testCase.filesOnDisk),
			}
			p.Init(testStore, newMockStoreUpdater(nil), func(loginp.Source) string { return "" })

			assert.ElementsMatch(t, testCase.expectedCleanedKeys, testStore.cleanedKeys)
		})
	}
}

func TestProspector_InitUpdateIdentifiers(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "existing_file")
	if err != nil {
		t.Fatalf("cannot create temp file")
	}
	defer f.Close()
	tmpFileName := f.Name()
	fi, err := f.Stat()
	if err != nil {
		t.Fatalf("cannot stat test file: %v", err)
	}

	testCases := map[string]struct {
		entries             map[string]loginp.Value
		filesOnDisk         map[string]loginp.FileDescriptor
		expectedUpdatedKeys map[string]string
		newKey              string
	}{
		"prospector init does not update keys if there are no entries": {
			entries:             nil,
			filesOnDisk:         nil,
			expectedUpdatedKeys: map[string]string{},
		},
		"prospector init does not update keys of not existing files": {
			entries: map[string]loginp.Value{
				"not_path::key1": &mockUnpackValue{
					key: "not_path::key1",
					fileMeta: fileMeta{
						Source:         "/no/such/path",
						IdentifierName: "not_path",
					},
				},
			},
			filesOnDisk:         nil,
			expectedUpdatedKeys: map[string]string{},
		},
		"prospector init does not update keys if new file identity is not fingerprint": {
			entries: map[string]loginp.Value{
				"not_path::key1": &mockUnpackValue{
					key: "not_path::key1",
					fileMeta: fileMeta{
						Source:         tmpFileName,
						IdentifierName: "not_path",
					},
				},
			},
			filesOnDisk: map[string]loginp.FileDescriptor{
				tmpFileName: {Info: file.ExtendFileInfo(fi)},
			},
			expectedUpdatedKeys: map[string]string{},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			testStore := newMockStoreUpdater(testCase.entries)
			p := fileProspector{
				logger:      logp.NewNopLogger(),
				identifier:  mustPathIdentifier(false),
				filewatcher: newMockFileWatcherWithFiles(testCase.filesOnDisk),
			}
			err := p.Init(testStore, newMockStoreUpdater(nil), func(loginp.Source) string { return testCase.newKey })
			require.NoError(t, err, "prospector Init must succeed")
			assert.Equal(t, testCase.expectedUpdatedKeys, testStore.updatedKeys)
		})
	}
}

func TestProspector_UpdateIdentifiersOnlyForSameFiles(t *testing.T) {
	workDir := testingfs.TempDir(t, "")
	sourcePath := filepath.Join(workDir, "app.log")
	oldSourcePath := filepath.Join(workDir, "app.log.1")

	oldFile, err := os.Create(oldSourcePath)
	require.NoError(t, err, "creating old log file")
	defer oldFile.Close()

	oldInfo, err := oldFile.Stat()
	require.NoError(t, err, "stating old log file")
	oldDescriptor := loginp.FileDescriptor{
		Filename:    sourcePath,
		Info:        file.ExtendFileInfo(oldInfo),
		Fingerprint: loginp.FingerprintID{Sum: "old-fingerprint"},
	}

	currentFile, err := os.Create(sourcePath)
	require.NoError(t, err, "creating current log file")
	defer currentFile.Close()

	currentInfo, err := currentFile.Stat()
	require.NoError(t, err, "stating current log file")
	currentDescriptor := loginp.FileDescriptor{
		Filename:    sourcePath,
		Info:        file.ExtendFileInfo(currentInfo),
		Fingerprint: loginp.FingerprintID{Sum: "current-fingerprint"},
	}

	globalIdentifier, err := loginp.NewSourceIdentifier(pluginName, "")
	require.NoError(t, err, "creating global source identifier")
	inputIdentifier, err := loginp.NewSourceIdentifier(pluginName, "input-id")
	require.NoError(t, err, "creating input source identifier")

	for _, identityName := range []string{nativeName, fingerprintName} {
		t.Run(identityName, func(t *testing.T) {
			identifier := mustIdentifier(t, identityName)
			oldSource := identifier.GetSource(loginp.FSEvent{
				NewPath:    sourcePath,
				Descriptor: oldDescriptor,
			})

			oldKey := globalIdentifier.ID(oldSource)
			globalStore := newMockStoreUpdater(map[string]loginp.Value{
				oldKey: &mockUnpackValue{
					key: oldKey,
					fileMeta: fileMeta{
						Source:         sourcePath,
						IdentifierName: identityName,
					},
				},
			})

			p := fileProspector{
				logger:     logptest.NewFileLogger(t, workDir).Logger,
				identifier: identifier,
				filewatcher: newMockFileWatcherWithFiles(
					map[string]loginp.FileDescriptor{
						sourcePath: currentDescriptor,
					}),
			}

			err = p.Init(newMockStoreUpdater(nil), globalStore, inputIdentifier.ID)
			require.NoError(t, err, "prospector Init must succeed")
			assert.Empty(
				t,
				globalStore.updatedKeys,
				"stale global registry entry must not be migrated to the current file",
			)
		})
	}
}

func TestMigrateRegistryToFingerprint(t *testing.T) {
	const mockFingerprint = "the fingerprint from this file"
	const mockInputPrefix = "test-input"

	logFileFullPath, err := filepath.Abs(filepath.Join("testdata", "log.log"))
	if err != nil {
		t.Fatalf("cannot get absolute path from test file: %s", err)
	}
	f, err := os.Open(logFileFullPath)
	if err != nil {
		t.Fatalf("cannot open test file")
	}
	defer f.Close()
	tmpFileName := f.Name()
	fi, err := f.Stat()

	fd := loginp.FileDescriptor{
		Filename:    tmpFileName,
		Info:        file.ExtendFileInfo(fi),
		Fingerprint: loginp.FingerprintID{Sum: mockFingerprint},
	}

	fingerprintIdentifier, _ := newFingerprintIdentifier(nil, nil)
	nativeIdentifier, _ := newINodeDeviceIdentifier(nil, nil)
	pathIdentifier, _ := newPathIdentifier(nil, nil)
	newIDFunc := func(s loginp.Source) string {
		return mockInputPrefix + "-" + s.Name()
	}

	fsEvent := loginp.FSEvent{
		OldPath:    logFileFullPath,
		NewPath:    logFileFullPath,
		Op:         loginp.OpCreate,
		Descriptor: fd,
	}

	expectedNewKey := newIDFunc(fingerprintIdentifier.GetSource(fsEvent))

	testCases := map[string]struct {
		oldIdentifier           fileIdentifier
		newIdentifier           fileIdentifier
		expectRegistryMigration bool
	}{
		"inode to fingerprint succeeds": {
			oldIdentifier:           nativeIdentifier,
			newIdentifier:           fingerprintIdentifier,
			expectRegistryMigration: true,
		},
		"path to fingerprint succeeds": {
			oldIdentifier:           pathIdentifier,
			newIdentifier:           fingerprintIdentifier,
			expectRegistryMigration: true,
		},
		"fingerprint to fingerprint fails": {
			oldIdentifier: fingerprintIdentifier,
			newIdentifier: fingerprintIdentifier,
		},

		// If the new identifier is not fingerprint, it will always fail.
		// So we only test a couple of combinations
		"fingerprint to native fails": {
			oldIdentifier: fingerprintIdentifier,
			newIdentifier: nativeIdentifier,
		},
		"path to native fails": {
			oldIdentifier: pathIdentifier,
			newIdentifier: nativeIdentifier,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			oldKey := newIDFunc(tc.oldIdentifier.GetSource(fsEvent))
			entries := map[string]loginp.Value{
				oldKey: &mockUnpackValue{
					key: oldKey,
					fileMeta: fileMeta{
						Source:         logFileFullPath,
						IdentifierName: tc.oldIdentifier.Name(),
					},
				},
			}

			testStore := newMockStoreUpdater(entries)
			filesOnDisk := map[string]loginp.FileDescriptor{
				tmpFileName: fd,
			}

			p := fileProspector{
				logger:      logp.NewNopLogger(),
				identifier:  tc.newIdentifier,
				filewatcher: newMockFileWatcherWithFiles(filesOnDisk),
			}

			err = p.Init(
				testStore,
				newMockStoreUpdater(nil),
				newIDFunc,
			)
			require.NoError(t, err, "prospector Init must succeed")

			// testStore.updatedKeys is in the format
			// oldKey -> newKey
			if tc.expectRegistryMigration {
				assert.Equal(
					t,
					map[string]string{
						oldKey: expectedNewKey,
					},
					testStore.updatedKeys,
					"the registry entries were not correctly migrated")
			} else {
				assert.Equal(
					t,
					map[string]string{},
					testStore.updatedKeys,
					"expecting no migration")
			}
		})
	}
}

func TestProspectorNewAndUpdatedFiles(t *testing.T) {
	minuteAgo := time.Now().Add(-1 * time.Minute)

	testCases := map[string]struct {
		events         []loginp.FSEvent
		ignoreOlder    time.Duration
		expectedEvents []harvesterEvent
	}{
		"two new files": {
			events: []loginp.FSEvent{
				{Op: loginp.OpCreate, NewPath: "/path/to/file", Descriptor: createTestFileDescriptor()},
				{Op: loginp.OpCreate, NewPath: "/path/to/other/file", Descriptor: createTestFileDescriptor()},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/file"),
				harvesterStart("path::/path/to/other/file"),
				harvesterGroupStop{},
			},
		},
		"one updated file": {
			events: []loginp.FSEvent{
				{Op: loginp.OpWrite, NewPath: "/path/to/file", Descriptor: createTestFileDescriptor()},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/file"),
				harvesterGroupStop{},
			},
		},
		"one updated then truncated file": {
			events: []loginp.FSEvent{
				{Op: loginp.OpWrite, NewPath: "/path/to/file", Descriptor: createTestFileDescriptor()},
				{Op: loginp.OpTruncate, NewPath: "/path/to/file", Descriptor: createTestFileDescriptor()},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/file"),
				harvesterRestart("path::/path/to/file"),
				harvesterGroupStop{},
			},
		},
		"old files with ignore older configured": {
			events: []loginp.FSEvent{
				{
					Op:         loginp.OpCreate,
					NewPath:    "/path/to/file",
					Descriptor: createTestFileDescriptorWithInfo(&testFileInfo{"/path/to/file", 5, minuteAgo, nil}),
				},
				{
					Op:         loginp.OpWrite,
					NewPath:    "/path/to/other/file",
					Descriptor: createTestFileDescriptorWithInfo(&testFileInfo{"/path/to/other/file", 5, minuteAgo, nil}),
				},
			},
			ignoreOlder: 10 * time.Second,
			expectedEvents: []harvesterEvent{
				harvesterGroupStop{},
			},
		},
		"newer files with ignore older": {
			events: []loginp.FSEvent{
				{
					Op:         loginp.OpCreate,
					NewPath:    "/path/to/file",
					Descriptor: createTestFileDescriptorWithInfo(&testFileInfo{"/path/to/file", 5, minuteAgo, nil}),
				},
				{
					Op:         loginp.OpWrite,
					NewPath:    "/path/to/other/file",
					Descriptor: createTestFileDescriptorWithInfo(&testFileInfo{"/path/to/other/file", 5, minuteAgo, nil}),
				},
			},
			ignoreOlder: 5 * time.Minute,
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/file"),
				harvesterStart("path::/path/to/other/file"),
				harvesterGroupStop{},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			p := fileProspector{
				logger:      logp.NewNopLogger(),
				filewatcher: newMockFileWatcher(test.events, len(test.events)),
				identifier:  mustPathIdentifier(false),
				ignoreOlder: test.ignoreOlder,
			}
			ctx := input.Context{Logger: logp.NewNopLogger(), Cancelation: context.Background()}
			hg := newTestHarvesterGroup()

			p.Run(ctx, newMockMetadataUpdater(), hg)

			assert.ElementsMatch(t, test.expectedEvents, hg.events, "expected different harvester events")
		})
	}
}

// TestProspectorHarvesterUpdateIgnoredFiles checks if the prospector can
// save the size of an ignored file to the registry. If the ignored
// file is updated, and has to be collected, a new harvester is started.
func TestProspectorHarvesterUpdateIgnoredFiles(t *testing.T) {
	minuteAgo := time.Now().Add(-1 * time.Minute)

	eventCreate := loginp.FSEvent{
		Op:         loginp.OpCreate,
		NewPath:    "/path/to/file",
		Descriptor: createTestFileDescriptorWithInfo(&testFileInfo{"/path/to/file", 5, minuteAgo, nil}),
	}
	eventUpdated := loginp.FSEvent{
		Op:         loginp.OpWrite,
		NewPath:    "/path/to/file",
		Descriptor: createTestFileDescriptorWithInfo(&testFileInfo{"/path/to/file", 10, time.Now(), nil}),
	}
	expectedEvents := []harvesterEvent{
		harvesterStart("path::/path/to/file"),
		harvesterGroupStop{},
	}

	filewatcher := newMockFileWatcher([]loginp.FSEvent{eventCreate}, 2)
	p := fileProspector{
		logger:      logp.NewNopLogger(),
		filewatcher: filewatcher,
		identifier:  mustPathIdentifier(false),
		ignoreOlder: 10 * time.Second,
	}
	ctx := input.Context{Logger: logp.NewNopLogger(), Cancelation: context.Background()}
	hg := newTestHarvesterGroup()
	testStore := newMockMetadataUpdater()
	var wg sync.WaitGroup
	wg.Go(func() {
		p.Run(ctx, testStore, hg)

	})

	// The prospector must persist the size of the file to the state
	// as the offset, so when the file is updated only the new
	// lines are sent to the output.
	assert.Eventually(
		t,
		func() bool { return testStore.checkOffset("path::/path/to/file", 5) },
		1*time.Second,
		10*time.Millisecond,
		"file state has to be persisted",
	)

	// The ignored file is updated, so the prospector must start a new harvester
	// to read the new lines.
	filewatcher.out <- eventUpdated
	wg.Wait()

	assert.Eventually(
		t,
		func() bool { return assert.ElementsMatch(t, expectedEvents, hg.events) },
		1*time.Second,
		10*time.Millisecond,
		"expected different harvester events",
	)
}

func TestProspectorDeletedFile(t *testing.T) {
	testCases := map[string]struct {
		events       []loginp.FSEvent
		cleanRemoved bool
	}{
		"one deleted file without clean removed": {
			events: []loginp.FSEvent{
				{Op: loginp.OpDelete, OldPath: "/path/to/file", Descriptor: createTestFileDescriptor()},
			},
			cleanRemoved: false,
		},
		"one deleted file with clean removed": {
			events: []loginp.FSEvent{
				{Op: loginp.OpDelete, OldPath: "/path/to/file", Descriptor: createTestFileDescriptor()},
			},
			cleanRemoved: true,
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			p := fileProspector{
				logger:       logp.NewNopLogger(),
				filewatcher:  newMockFileWatcher(test.events, len(test.events)),
				identifier:   mustPathIdentifier(false),
				cleanRemoved: test.cleanRemoved,
			}
			ctx := input.Context{Logger: logp.NewNopLogger(), Cancelation: context.Background()}

			testStore := newMockMetadataUpdater()
			testStore.set("path::/path/to/file")

			p.Run(ctx, testStore, newTestHarvesterGroup())

			has := testStore.has("path::/path/to/file")

			if test.cleanRemoved {
				assert.False(t, has)
			} else {
				assert.True(t, has)
			}
		})
	}
}

func TestProspectorRenamedFile(t *testing.T) {
	testCases := map[string]struct {
		events         []loginp.FSEvent
		trackRename    bool
		closeRenamed   bool
		expectedEvents []harvesterEvent
	}{
		"one renamed file without rename tracker": {
			events: []loginp.FSEvent{
				{
					Op:         loginp.OpRename,
					OldPath:    "/old/path/to/file",
					NewPath:    "/new/path/to/file",
					Descriptor: createTestFileDescriptor(),
				},
			},
			expectedEvents: []harvesterEvent{
				harvesterStop("path::/old/path/to/file"),
				harvesterStart("path::/new/path/to/file"),
				harvesterGroupStop{},
			},
		},
		"one renamed file with rename tracker": {
			events: []loginp.FSEvent{
				{
					Op:         loginp.OpRename,
					OldPath:    "/old/path/to/file",
					NewPath:    "/new/path/to/file",
					Descriptor: createTestFileDescriptor(),
				},
			},
			trackRename: true,
			expectedEvents: []harvesterEvent{
				harvesterGroupStop{},
			},
		},
		"one renamed file with rename tracker with close renamed": {
			events: []loginp.FSEvent{
				{
					Op:         loginp.OpRename,
					OldPath:    "/old/path/to/file",
					NewPath:    "/new/path/to/file",
					Descriptor: createTestFileDescriptor(),
				},
			},
			trackRename:  true,
			closeRenamed: true,
			expectedEvents: []harvesterEvent{
				harvesterStop("path::/old/path/to/file"),
				harvesterGroupStop{},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			p := fileProspector{
				logger:            logp.NewNopLogger(),
				filewatcher:       newMockFileWatcher(test.events, len(test.events)),
				identifier:        mustPathIdentifier(test.trackRename),
				stateChangeCloser: stateChangeCloserConfig{Renamed: test.closeRenamed},
			}
			ctx := input.Context{Logger: logp.NewNopLogger(), Cancelation: context.Background()}

			testStore := newMockMetadataUpdater()
			testStore.set("path::/old/path/to/file")

			hg := newTestHarvesterGroup()
			p.Run(ctx, testStore, hg)

			has := testStore.has("path::/old/path/to/file")
			if test.trackRename {
				assert.True(t, has)
			} else {
				assert.False(t, has)
			}

			assert.Equal(t, test.expectedEvents, hg.events)
		})
	}
}

type harvesterEvent interface{ String() string }

type harvesterStart string

func (h harvesterStart) String() string { return string(h) }

type harvesterRestart string

func (h harvesterRestart) String() string { return string(h) }

type harvesterContinue string

func (h harvesterContinue) String() string { return string(h) }

type harvesterStop string

func (h harvesterStop) String() string { return string(h) }

type harvesterGroupStop struct{}

func (h harvesterGroupStop) String() string { return "stop" }

type testHarvesterGroup struct {
	events []harvesterEvent
}

func newTestHarvesterGroup() *testHarvesterGroup {
	return &testHarvesterGroup{make([]harvesterEvent, 0)}
}

func (t *testHarvesterGroup) Start(_ input.Context, s loginp.Source) {
	t.events = append(t.events, harvesterStart(s.Name()))
}

func (t *testHarvesterGroup) Restart(_ input.Context, s loginp.Source) {
	t.events = append(t.events, harvesterRestart(s.Name()))
}

func (t *testHarvesterGroup) Continue(_ input.Context, p, s loginp.Source) {
	t.events = append(t.events, harvesterContinue(p.Name()+" -> "+s.Name()))
}

func (t *testHarvesterGroup) Stop(s loginp.Source) {
	t.events = append(t.events, harvesterStop(s.Name()))
}

func (t *testHarvesterGroup) StopHarvesters() error {
	t.events = append(t.events, harvesterGroupStop{})
	return nil
}

// SetObserver is a no-op
func (t *testHarvesterGroup) SetObserver(c chan loginp.HarvesterStatus) {
}

type mockFileWatcher struct {
	events      []loginp.FSEvent
	filesOnDisk map[string]loginp.FileDescriptor

	outputCount, eventCount int

	out chan loginp.FSEvent

	c chan loginp.HarvesterStatus
}

// newMockFileWatcher creates an FSWatch mock, so you can read
// the required FSEvents from it using the Event function.
func newMockFileWatcher(events []loginp.FSEvent, eventCount int) *mockFileWatcher {
	w := &mockFileWatcher{
		events:     events,
		eventCount: eventCount,
		out:        make(chan loginp.FSEvent, eventCount),
		c:          make(chan loginp.HarvesterStatus),
	}

	for _, evt := range events {
		w.out <- evt
	}

	return w
}

// newMockFileWatcherWithFiles creates an FSWatch mock to
// get the required file information from the file system using
// the GetFiles function.
func newMockFileWatcherWithFiles(filesOnDisk map[string]loginp.FileDescriptor) *mockFileWatcher {
	return &mockFileWatcher{
		filesOnDisk: filesOnDisk,
		out:         make(chan loginp.FSEvent),
	}
}

func (m *mockFileWatcher) Event() loginp.FSEvent {
	if m.outputCount == m.eventCount {
		close(m.out)
		return loginp.FSEvent{}
	}
	evt := <-m.out
	m.outputCount = m.outputCount + 1
	return evt
}

func (m *mockFileWatcher) Run(_ unison.Canceler) {}

func (m *mockFileWatcher) GetFiles() map[string]loginp.FileDescriptor { return m.filesOnDisk }

func (m *mockFileWatcher) NotifyChan() chan loginp.HarvesterStatus {
	return m.c
}

// mockMetadataUpdater is a test implementation of loginp.MetadataUpdater whose
// methods may be invoked from the prospector's goroutines while the test
// goroutine inspects the stored state (e.g. via assert.Eventually). Read paths
// dominate (assert.Eventually polls), so an RWMutex is used to allow
// concurrent reads.
type mockMetadataUpdater struct {
	mu    sync.RWMutex
	table map[string]any

	FindCursorMetaCalled  atomic.Int64
	ResetCursorCalled     int
	UpdateMetadataCalled  int
	RemoveCalled          int
	IterateOnPrefixCalled atomic.Int64
	KeyExistsCalled       atomic.Int64
	UpdateKeyCalled       int
}

func newMockMetadataUpdater() *mockMetadataUpdater {
	return &mockMetadataUpdater{
		table: make(map[string]any),
	}
}

func (mu *mockMetadataUpdater) set(id string) {
	mu.mu.Lock()
	defer mu.mu.Unlock()
	mu.table[id] = struct{}{}
}

// setRaw stores an arbitrary value under id. Used by tests that pre-populate
// the store before running the prospector.
func (mu *mockMetadataUpdater) setRaw(id string, v any) {
	mu.mu.Lock()
	defer mu.mu.Unlock()
	mu.table[id] = v
}

// get returns the raw value stored under id. Used by tests that need to
// inspect the stored value after the prospector has run.
func (mu *mockMetadataUpdater) get(id string) any {
	mu.mu.RLock()
	defer mu.mu.RUnlock()
	return mu.table[id]
}

func (mu *mockMetadataUpdater) has(id string) bool {
	mu.mu.RLock()
	defer mu.mu.RUnlock()
	_, ok := mu.table[id]
	return ok
}

func (mu *mockMetadataUpdater) checkOffset(id string, offset int64) bool {
	mu.mu.RLock()
	defer mu.mu.RUnlock()
	c, ok := mu.table[id]
	if !ok {
		return false
	}
	cursor, ok := c.(state)
	if !ok {
		return false
	}
	return cursor.Offset == offset
}

func (mu *mockMetadataUpdater) FindCursorMeta(s loginp.Source, v any) error {
	mu.mu.RLock()
	defer mu.mu.RUnlock()
	mu.FindCursorMetaCalled.Add(1)
	meta, ok := mu.table[s.Name()]
	if !ok {
		return fmt.Errorf("no such id [%q]", s.Name())
	}
	return typeconv.Convert(v, meta)
}

func (mu *mockMetadataUpdater) ResetCursor(s loginp.Source, cur any) error {
	mu.mu.Lock()
	defer mu.mu.Unlock()
	mu.ResetCursorCalled++
	mu.table[s.Name()] = cur
	return nil
}

func (mu *mockMetadataUpdater) UpdateMetadata(s loginp.Source, v any) error {
	mu.mu.Lock()
	defer mu.mu.Unlock()
	mu.UpdateMetadataCalled++
	mu.table[s.Name()] = v
	return nil
}

func (mu *mockMetadataUpdater) Remove(s loginp.Source) error {
	mu.mu.Lock()
	defer mu.mu.Unlock()
	mu.RemoveCalled++
	delete(mu.table, s.Name())
	return nil
}

func (mu *mockMetadataUpdater) IterateOnPrefix(fn func(key string, meta any)) {
	mu.mu.RLock()
	defer mu.mu.RUnlock()
	mu.IterateOnPrefixCalled.Add(1)
	for key, meta := range mu.table {
		fn(key, meta)
	}
}

func (mu *mockMetadataUpdater) KeyExists(key string) bool {
	mu.mu.RLock()
	defer mu.mu.RUnlock()
	mu.KeyExistsCalled.Add(1)
	_, ok := mu.table[key]
	return ok
}

func (mu *mockMetadataUpdater) UpdateKey(oldKey, newKey string, meta any) error {
	mu.mu.Lock()
	defer mu.mu.Unlock()
	mu.UpdateKeyCalled++
	if _, ok := mu.table[oldKey]; !ok {
		return fmt.Errorf("old key %s not found", oldKey)
	}
	mu.table[newKey] = meta
	delete(mu.table, oldKey)
	return nil
}

type mockUnpackValue struct {
	fileMeta
	key string
}

func (u *mockUnpackValue) UnpackCursorMeta(to any) error {
	return typeconv.Convert(to, u.fileMeta)
}

func (u *mockUnpackValue) Key() string {
	return u.key
}

type mockStoreUpdater struct {
	available   map[string]loginp.Value
	cleanedKeys []string
	updatedKeys map[string]string
}

func newMockStoreUpdater(available map[string]loginp.Value) *mockStoreUpdater {
	return &mockStoreUpdater{
		available:   available,
		cleanedKeys: make([]string, 0),
		updatedKeys: make(map[string]string, 0),
	}
}

func (m *mockStoreUpdater) CleanIf(pred func(v loginp.Value) bool) {
	for key, meta := range m.available {
		if pred(meta) {
			m.cleanedKeys = append(m.cleanedKeys, key)
		}
	}
}

func (m *mockStoreUpdater) UpdateIdentifiers(updater func(v loginp.Value) (string, any)) {
	for key, meta := range m.available {
		k, _ := updater(meta)
		if k != "" {
			m.updatedKeys[key] = k
		}
	}
}

// TakeOver is a noop on this mock
func (m *mockStoreUpdater) TakeOver(func(v loginp.TakeOverState) (string, any)) {}

type renamedPathIdentifier struct {
	fileIdentifier
}

func (p *renamedPathIdentifier) Supports(_ identifierFeature) bool { return true }

func mustPathIdentifier(renamed bool) fileIdentifier {
	pathIdentifier, err := newPathIdentifier(nil, logp.NewNopLogger())
	if err != nil {
		panic(err)
	}
	if renamed {
		return &renamedPathIdentifier{pathIdentifier}
	}
	return pathIdentifier
}

func TestOnRenameFileIdentity(t *testing.T) {
	testCases := map[string]struct {
		identifier    string
		events        []loginp.FSEvent
		populateStore bool
		errMsg        string
	}{
		"identifier name from meta is kept": {
			identifier:    "foo",
			errMsg:        "must be the same as in the registry",
			populateStore: true,
			events: []loginp.FSEvent{
				{
					Op:         loginp.OpRename,
					OldPath:    "/old/path/to/file",
					NewPath:    "/new/path/to/file",
					Descriptor: createTestFileDescriptor(),
				},
			},
		},
		"identifier from prospector is used": {
			identifier:    "path",
			errMsg:        "must come from prospector configuration",
			populateStore: false,
			events: []loginp.FSEvent{
				{
					Op:         loginp.OpRename,
					OldPath:    "/old/path/to/file",
					NewPath:    "/new/path/to/file",
					Descriptor: createTestFileDescriptor(),
				},
			},
		},
	}

	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			p := fileProspector{
				logger:            logp.NewNopLogger(),
				filewatcher:       newMockFileWatcher(tc.events, len(tc.events)),
				identifier:        mustPathIdentifier(true),
				stateChangeCloser: stateChangeCloserConfig{Renamed: true},
			}
			ctx := input.Context{Logger: logp.NewNopLogger(), Cancelation: context.Background()}

			path := "/new/path/to/file"
			expectedIdentifier := tc.identifier
			id := "path" + "::" + path

			testStore := newMockMetadataUpdater()
			if tc.populateStore {
				testStore.setRaw(id, fileMeta{Source: path, IdentifierName: expectedIdentifier})
			}

			hg := newTestHarvesterGroup()
			p.Run(ctx, testStore, hg)

			got := testStore.get(id)
			meta := fileMeta{}
			typeconv.Convert(&meta, got)

			if meta.IdentifierName != expectedIdentifier {
				t.Errorf("fileMeta.IdentifierName %s, expecting: %q, got: %q", tc.errMsg, expectedIdentifier, meta.IdentifierName)
			}
		})
	}
}

type testFileInfo struct {
	name string
	size int64
	time time.Time
	sys  any
}

func (t *testFileInfo) Name() string       { return t.name }
func (t *testFileInfo) Size() int64        { return t.size }
func (t *testFileInfo) Mode() os.FileMode  { return 0 }
func (t *testFileInfo) ModTime() time.Time { return t.time }
func (t *testFileInfo) IsDir() bool        { return false }
func (t *testFileInfo) Sys() any           { return t.sys }

func createTestFileDescriptor() loginp.FileDescriptor {
	return createTestFileDescriptorWithInfo(&testFileInfo{})
}

func createTestFileDescriptorWithInfo(fi fs.FileInfo) loginp.FileDescriptor {
	return loginp.FileDescriptor{
		Info:        file.ExtendFileInfo(fi),
		Fingerprint: loginp.FingerprintID{Sum: "fingerprint"},
		Filename:    "filename",
	}
}

func TestFileProspector_previousID(t *testing.T) {
	testFileInfo := &testFileInfo{
		name: "/path/to/file",
		size: 100,
		time: time.Now(),
		sys:  nil,
	}
	fd := loginp.FileDescriptor{
		Filename:    "/path/to/file",
		Info:        file.ExtendFileInfo(testFileInfo),
		Fingerprint: loginp.FingerprintID{Sum: "test-fingerprint"},
	}

	tests := map[string]struct {
		takeOverConfig loginp.TakeOverConfig
		identifierName string
		takeOverState  loginp.TakeOverState
		validateID     func(t *testing.T, id string)
	}{
		"from filestream - native identifier": {
			takeOverConfig: loginp.TakeOverConfig{
				Enabled: true,
				FromIDs: []string{"some-id"},
			},
			identifierName: nativeName,
			takeOverState: loginp.TakeOverState{
				Source:         "/path/to/file",
				IdentifierName: nativeName,
			},
			validateID: func(t *testing.T, id string) {
				// native identifier is OS specific, so we cannot hard code the expected result
				assert.Contains(t, id, nativeName+identitySep, "ID should contain native identifier prefix")
				assert.Contains(t, id, fd.Info.GetOSState().Identifier(), "ID should contain OS identifier")
			},
		},
		"from filestream - path identifier": {
			takeOverConfig: loginp.TakeOverConfig{
				Enabled: true,
				FromIDs: []string{"some-id"},
			},
			identifierName: pathName,
			takeOverState: loginp.TakeOverState{
				Source:         "/path/to/file",
				IdentifierName: pathName,
			},
			validateID: func(t *testing.T, id string) {
				assert.Equal(t, pathName+identitySep+"/path/to/file", id, "ID should match path identifier format")
			},
		},
		"from filestream input with stderr stream": {
			takeOverConfig: loginp.TakeOverConfig{
				Enabled: true,
				FromIDs: []string{"some-id"},
				Stream:  "stderr",
			},
			identifierName: nativeName,
			takeOverState: loginp.TakeOverState{
				Source: "/path/to/file",
				Key:    "test-key",
			},
			validateID: func(t *testing.T, id string) {
				assert.Contains(t, id, nativeName+identitySep, "ID should contain native identifier prefix")
				if !strings.HasSuffix(id, "stderr") {
					t.Errorf("ID must end in 'stderr' (take_over.stream), got: %s", id)
				}
				assert.NotEqual(t, nativeName+identitySep+fd.Info.GetOSState().Identifier(), id, "ID with stream metadata should have hash prefix")
			},
		},
		"from log input - native identifier": {
			takeOverConfig: loginp.TakeOverConfig{
				Enabled: true,
			},
			identifierName: nativeName,
			takeOverState: loginp.TakeOverState{
				Source:      "/path/to/file",
				FileStateOS: fd.Info.GetOSState(),
			},
			validateID: func(t *testing.T, id string) {
				assert.Contains(t, id, nativeName+identitySep, "ID should contain native identifier prefix")
				assert.Contains(t, id, fd.Info.GetOSState().Identifier(), "ID should contain OS identifier")
			},
		},
		"from log input - path identifier": {
			takeOverConfig: loginp.TakeOverConfig{
				Enabled: true,
			},
			identifierName: pathName,
			takeOverState: loginp.TakeOverState{
				Source:      "/path/to/file",
				FileStateOS: fd.Info.GetOSState(),
				Key:         "test-key",
			},
			validateID: func(t *testing.T, id string) {
				assert.Equal(t, pathName+identitySep+"/path/to/file", id, "ID should match path identifier format")
			},
		},
		"from log input - native with stdout stream": {
			takeOverConfig: loginp.TakeOverConfig{
				Enabled: true,
				Stream:  "stdout",
			},
			identifierName: nativeName,
			takeOverState: loginp.TakeOverState{
				Source:      "/path/to/file",
				FileStateOS: fd.Info.GetOSState(),
			},
			validateID: func(t *testing.T, id string) {
				assert.Contains(t, id, nativeName+identitySep, "ID should contain native identifier prefix")
				assert.Containsf(t, id, "1b59052b95e61943", "ID should contain the meta hash '1b59052b95e61943'")
				assert.NotEqual(t, nativeName+identitySep+fd.Info.GetOSState().Identifier(), id, "ID with stream metadata should have hash prefix")
			},
		},
		"from log input with stderr stream": {
			takeOverConfig: loginp.TakeOverConfig{
				Enabled: true,
				Stream:  "stderr",
			},
			identifierName: nativeName,
			takeOverState: loginp.TakeOverState{
				Source:      "/path/to/file",
				FileStateOS: fd.Info.GetOSState(),
				Key:         "test-key",
			},
			validateID: func(t *testing.T, id string) {
				assert.Contains(t, id, nativeName+identitySep, "ID should contain native identifier prefix")
				assert.Containsf(t, id, "d35e05a633229937", "ID should contain the meta hash 'd35e05a633229937'")
				assert.NotEqual(t, nativeName+identitySep+fd.Info.GetOSState().Identifier(), id, "ID with stream metadata should have hash prefix")
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			p := &fileProspector{
				logger:                logp.NewNopLogger(),
				takeOver:              tc.takeOverConfig,
				filestreamIdentifiers: filestreamFileIdentifiers(logp.NewNopLogger(), tc.takeOverConfig.Stream),
				logIdentifiers:        logFileIdentifiers(logp.NewNopLogger()),
			}

			id := p.previousID(tc.identifierName, fd, tc.takeOverState)
			tc.validateID(t, id)
		})
	}
}

func TestFileProspector_takeOverFn(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("not supported on Windows because inode marker is used")
	}
	testFileInfo := &testFileInfo{
		name: "/path/to/file",
		size: 100,
		time: time.Now(),
		sys:  nil,
	}
	fd := loginp.FileDescriptor{
		Filename:    "/path/to/file",
		Info:        file.ExtendFileInfo(testFileInfo),
		Fingerprint: loginp.FingerprintID{Sum: "test-fingerprint"},
	}

	tests := map[string]struct {
		identifier     fileIdentifier
		takeOverState  loginp.TakeOverState
		files          map[string]loginp.FileDescriptor
		newIDFunc      func(loginp.Source) string
		expectedNewKey string
		expectedMeta   any
		shouldTakeOver bool
	}{
		"file not on disk": {
			identifier: mustIdentifier(t, pathName),
			takeOverState: loginp.TakeOverState{
				Source:         "/missing/file",
				IdentifierName: pathName,
				Key:            "filestream::test-id::path::/missing/file",
			},
			files:          map[string]loginp.FileDescriptor{},
			newIDFunc:      func(s loginp.Source) string { return "filestream::new-id::" + s.Name() },
			expectedNewKey: "",
			expectedMeta: fileMeta{
				Source:         "/missing/file",
				IdentifierName: pathName,
			},
			shouldTakeOver: false,
		},
		"unsupported old identifier": {
			identifier: mustInodeMarker(t),
			takeOverState: loginp.TakeOverState{
				Source:         "/path/to/file",
				IdentifierName: inodeMarkerName,
				Key:            "filestream::test-id::inode_marker::/path/to/file",
			},
			files: map[string]loginp.FileDescriptor{
				"/path/to/file": fd,
			},
			newIDFunc:      func(s loginp.Source) string { return "filestream::new-id::" + s.Name() },
			expectedNewKey: "",
			expectedMeta:   nil,
			shouldTakeOver: false,
		},
		"registry key format invalid": {
			identifier: mustIdentifier(t, pathName),
			takeOverState: loginp.TakeOverState{
				Source:         "/path/to/file",
				IdentifierName: pathName,
				Key:            "invalid::format",
			},
			files: map[string]loginp.FileDescriptor{
				"/path/to/file": fd,
			},
			newIDFunc:      func(s loginp.Source) string { return "filestream::new-id::" + s.Name() },
			expectedNewKey: "",
			expectedMeta: fileMeta{
				Source:         "/path/to/file",
				IdentifierName: pathName,
			},
			shouldTakeOver: false,
		},
		"previous ID does not match registry ID": {
			identifier: mustIdentifier(t, pathName),
			takeOverState: loginp.TakeOverState{
				Source:         "/path/to/file",
				IdentifierName: pathName,
				Key:            "filestream::test-id::path::/different/path",
			},
			files: map[string]loginp.FileDescriptor{
				"/path/to/file": fd,
			},
			newIDFunc:      func(s loginp.Source) string { return "filestream::new-id::" + s.Name() },
			expectedNewKey: "",
			expectedMeta: fileMeta{
				Source:         "/path/to/file",
				IdentifierName: pathName,
			},
			shouldTakeOver: false,
		},
		"successful takeover - native to fingerprint": {
			identifier: mustIdentifier(t, fingerprintName),
			takeOverState: loginp.TakeOverState{
				Source:         "/path/to/file",
				IdentifierName: nativeName,
				Key:            "filestream::test-id::native::" + fd.Info.GetOSState().Identifier(),
				FileStateOS:    fd.Info.GetOSState(),
			},
			files: map[string]loginp.FileDescriptor{
				"/path/to/file": fd,
			},
			newIDFunc: func(s loginp.Source) string { return "filestream::new-id::" + s.Name() },
			expectedNewKey: func() string {
				fingerprintIdent := mustIdentifier(t, fingerprintName)
				source := fingerprintIdent.GetSource(loginp.FSEvent{NewPath: "/path/to/file", Descriptor: fd})
				return "filestream::new-id::" + source.Name()
			}(),
			expectedMeta: fileMeta{
				Source:         "/path/to/file",
				IdentifierName: fingerprintName,
			},
			shouldTakeOver: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			p := &fileProspector{
				logger:     logp.NewNopLogger(),
				identifier: tc.identifier,
				takeOver: loginp.TakeOverConfig{
					Enabled: true,
				},
				filestreamIdentifiers: filestreamFileIdentifiers(logp.NewNopLogger(), ""),
				logIdentifiers:        logFileIdentifiers(logp.NewNopLogger()),
			}

			newKey, meta := p.takeOverFn(tc.takeOverState, tc.files, tc.newIDFunc)

			if tc.shouldTakeOver {
				assert.Equal(t, tc.expectedNewKey, newKey, "new key does not match expected")
				assert.Equal(t, tc.expectedMeta, meta, "returned meta does not match expected")
			} else {
				assert.Empty(t, newKey, "expected empty key for non-takeover")
				if tc.expectedMeta == nil {
					assert.Nil(t, meta, "expected nil meta")
				} else {
					assert.Equal(t, tc.expectedMeta, meta, "returned meta does not match expected")
				}
			}
		})
	}
}

// mustIdentifier creates a fileIdentifier or fails the test
func mustIdentifier(t *testing.T, name string) fileIdentifier {
	t.Helper()
	factory, ok := identifierFactories[name]
	require.True(t, ok, "identifier factory not found: %s", name)

	identifier, err := factory(nil, logp.NewNopLogger())
	require.NoError(t, err, "failed to create identifier: %s", name)

	return identifier
}

func TestFindGrowingFingerprintMatch(t *testing.T) {
	const currentPath = "/var/log/app.log"

	testCases := map[string]struct {
		storeEntries       map[string]any
		currentFingerprint string
		currentPath        string
		expectedKey        string
		expectedFound      bool
	}{
		"empty current fingerprint returns immediately": {
			storeEntries:       map[string]any{},
			currentFingerprint: "",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"valid prefix match": {
			storeEntries: map[string]any{
				"filestream::my-input::fingerprint::aabb": fileMeta{
					Source:         currentPath,
					IdentifierName: fingerprintName,
					Fingerprint:    "aabb",
				},
			},
			currentFingerprint: "aabbccdd",
			currentPath:        currentPath,
			expectedKey:        "filestream::my-input::fingerprint::aabb",
			expectedFound:      true,
		},
		"prefix match among entries for different paths": {
			storeEntries: map[string]any{
				"filestream::my-input::fingerprint::aa": fileMeta{
					Source:         "/other/file.log",
					IdentifierName: fingerprintName,
					Fingerprint:    "aa",
				},
				"filestream::my-input::fingerprint::aabb": fileMeta{
					Source:         currentPath,
					IdentifierName: fingerprintName,
					Fingerprint:    "aabb",
				},
			},
			currentFingerprint: "aabbccddee",
			currentPath:        currentPath,
			expectedKey:        "filestream::my-input::fingerprint::aabb",
			expectedFound:      true,
		},
		"skips non-fingerprint identity": {
			storeEntries: map[string]any{
				"filestream::my-input::native::abc123": fileMeta{
					Source:         currentPath,
					IdentifierName: nativeName,
				},
			},
			currentFingerprint: "aabbccdd",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"skips key with too many separators": {
			storeEntries: map[string]any{
				"filestream::my-input::fingerprint::aabb::extra": fileMeta{
					Source:         currentPath,
					IdentifierName: fingerprintName,
					Fingerprint:    "aabb",
				},
			},
			currentFingerprint: "aabbccdd",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"skips key with too few separators": {
			storeEntries: map[string]any{
				"filestream::malformed": fileMeta{
					Source:         currentPath,
					IdentifierName: fingerprintName,
					Fingerprint:    "aabb",
				},
			},
			currentFingerprint: "aabbccdd",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"skips empty stored fingerprint": {
			// With the bounded-key optimization a growing entry is identified by
			// a non-empty fileMeta.Fingerprint (the raw hex), not by the key tail.
			// An entry with an empty Fingerprint is treated as final and skipped.
			storeEntries: map[string]any{
				"filestream::my-input::fingerprint::aabb": fileMeta{
					Source:         currentPath,
					IdentifierName: fingerprintName,
					Fingerprint:    "",
				},
			},
			currentFingerprint: "aabbccdd",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"skips stored fingerprint longer than current": {
			storeEntries: map[string]any{
				"filestream::my-input::fingerprint::aabbccddee": fileMeta{
					Source:         currentPath,
					IdentifierName: fingerprintName,
					Fingerprint:    "aabbccddee",
				},
			},
			currentFingerprint: "aabb",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"skips stored fingerprint equal length to current": {
			storeEntries: map[string]any{
				"filestream::my-input::fingerprint::aabb": fileMeta{
					Source:         currentPath,
					IdentifierName: fingerprintName,
					Fingerprint:    "aabb",
				},
			},
			currentFingerprint: "aabb",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"skips non-prefix fingerprint": {
			storeEntries: map[string]any{
				"filestream::my-input::fingerprint::xxxx": fileMeta{
					Source:         currentPath,
					IdentifierName: fingerprintName,
					Fingerprint:    "xxxx",
				},
			},
			currentFingerprint: "aabbccdd",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"skips path mismatch": {
			// Path-agnostic fallback is restricted to the threshold-crossing
			// case (a completed FingerprintID). For ordinary same-format growth
			// the stored entry's Source must match currentPath; mismatched
			// sources are rejected to avoid confusing two distinct files with a
			// shared content prefix for renames of one another.
			storeEntries: map[string]any{
				"filestream::my-input::fingerprint::aabb": fileMeta{
					Source:         "/other/file.log",
					IdentifierName: fingerprintName,
					Fingerprint:    "aabb",
				},
			},
			currentFingerprint: "aabbccdd",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"single colon in input ID is not a separator": {
			storeEntries: map[string]any{
				"filestream::my:input::fingerprint::aabb": fileMeta{
					Source:         currentPath,
					IdentifierName: fingerprintName,
					Fingerprint:    "aabb",
				},
			},
			currentFingerprint: "aabbccdd",
			currentPath:        currentPath,
			expectedKey:        "filestream::my:input::fingerprint::aabb",
			expectedFound:      true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			store := newMockMetadataUpdater()
			maps.Copy(store.table, tc.storeEntries)

			p := &fileProspector{logger: logptest.NewTestingLogger(t, "")}
			event := loginp.FSEvent{
				NewPath: tc.currentPath,
				Descriptor: loginp.FileDescriptor{
					Fingerprint: loginp.FingerprintID{Raw: tc.currentFingerprint},
				},
			}
			key, found := p.findGrowingFingerprintMatch(store, event)

			assert.Equal(t, tc.expectedFound, found, "found mismatch")
			if tc.expectedFound {
				assert.Equal(t, tc.expectedKey, key, "key mismatch")
			} else {
				assert.Empty(t, key)
			}
		})
	}
}

func TestHandleGrowingFingerprintLookup_KeyExistsFastPath(t *testing.T) {
	const currentFingerprint = "aabbccdd"
	const currentPath = "/var/log/app.log"

	identifier, err := newFingerprintIdentifier(nil, nil)
	require.NoError(t, err)

	event := loginp.FSEvent{
		NewPath: currentPath,
		SrcID:   "filestream::my-input::fingerprint::" + currentFingerprint,
		Descriptor: loginp.FileDescriptor{
			Fingerprint: loginp.FingerprintID{Raw: currentFingerprint},
		},
	}

	src := identifier.GetSource(event)

	t.Run("fast path: key exists skips scan", func(t *testing.T) {
		store := newMockMetadataUpdater()
		// The store already has an entry for the exact current fingerprint key.
		store.table[event.SrcID] = fileMeta{
			Source:         currentPath,
			IdentifierName: fingerprintName,
		}

		// Also add an entry that would be a prefix match (the slow path).
		// If the fast path works, it should never reach findGrowingFingerprintMatch.
		store.table["filestream::my-input::fingerprint::aabb"] = fileMeta{
			Source:         currentPath,
			IdentifierName: fingerprintName,
		}

		p := &fileProspector{
			logger:     logptest.NewTestingLogger(t, ""),
			identifier: identifier,
		}

		p.handleGrowingFingerprintLookup(logptest.NewTestingLogger(t, ""), event, src, store)

		// The fast path returns as soon as the exact key is found, so it must
		// never scan the registry or migrate anything.
		assert.Equal(t, int64(0), store.IterateOnPrefixCalled.Load(), "fast path must not scan the registry")
		assert.Equal(t, 0, store.UpdateKeyCalled, "fast path must not migrate")
		assert.Positive(t, store.KeyExistsCalled.Load(), "fast path must check key existence")
		// The pre-seeded prefix entry must remain untouched.
		assert.True(t, store.has("filestream::my-input::fingerprint::aabb"),
			"prefix entry must remain since migration did not run")
	})

	t.Run("slow path: key does not exist falls through to scan", func(t *testing.T) {
		const oldKey = "filestream::my-input::fingerprint::aabb"
		store := newMockMetadataUpdater()
		// Only a prefix match exists, not the exact key. The stored entry must
		// carry a real growing-phase raw fingerprint ("aabb") that is a strict
		// prefix of the event's raw fingerprint ("aabbccdd") so that
		// buildShortFingerprintSet indexes it and the prefix match succeeds.
		store.table[oldKey] = fileMeta{
			Source:         currentPath,
			IdentifierName: fingerprintName,
			Fingerprint:    "aabb",
		}

		p := &fileProspector{
			logger:     logptest.NewTestingLogger(t, ""),
			identifier: identifier,
		}

		p.handleGrowingFingerprintLookup(logptest.NewTestingLogger(t, ""), event, src, store)

		// Migration succeeds via the short fingerprint set prefix match: the
		// registry is scanned, the old key is migrated to the new identity, and
		// the old key is removed.
		assert.Positive(t, store.IterateOnPrefixCalled.Load(), "slow path must scan the registry")
		assert.Positive(t, store.UpdateKeyCalled, "slow path must migrate the matched entry")
		assert.False(t, store.has(oldKey), "old key must be removed after migration")
		// migrateGrowingFingerprint keeps the old key's plugin/input prefix and
		// swaps in the new identity (src.Name()), which is the SHA-256-derived
		// key, not the literal raw value used for event.SrcID.
		newKey := "filestream::my-input::" + src.Name()
		assert.NotEqual(t, oldKey, newKey)
		assert.True(t, store.has(newKey), "migrated entry must exist under the new key")
	})
}

// TestOnFSEvent_GrowingFingerprintMigration verifies that a write event whose
// fingerprint matches an existing growing entry triggers a store key migration.
// Two scenarios are covered:
//
//   - below-threshold growth: the descriptor still carries a raw-hex Raw
//     (Complete=false). The registry key migrates from the shorter raw-hex to
//     the longer raw-hex; the resulting state still carries the raw fingerprint
//     so the new entry remains in the short-fingerprint index for further
//     matching.
//   - at-threshold transition: the descriptor is Complete (Sum holds the
//     SHA-256) and still carries the raw header in Raw for one scan. The
//     registry key migrates to the SHA-256 key; the resulting state has an
//     empty raw fingerprint (omitted from the serialized form) and is dropped
//     from the short-fingerprint index.
func TestOnFSEvent_GrowingFingerprintMigration(t *testing.T) {
	path := "/var/log/app.log"
	oldFingerprint := "aabb"
	inputID := "my-input"
	log := logptest.NewTestingLogger(t, "")

	oldKey := "filestream::" + inputID + "::fingerprint::" + oldFingerprint

	identifier, err := newFingerprintIdentifier(nil, nil)
	require.NoError(t, err, "newFingerprintIdentifier failed")

	t.Run("below threshold: raw-hex grows to longer raw-hex", func(t *testing.T) {
		newFingerprint := oldFingerprint + "ccdd"
		desc := loginp.FileDescriptor{
			Fingerprint: loginp.FingerprintID{Raw: newFingerprint},
		}
		// With the bounded-key optimization the migrated key is a hash of the
		// raw fingerprint, not the raw fingerprint itself.
		newKey := "filestream::" + inputID + "::fingerprint::" + desc.Fingerprint.Key()

		store := newMockMetadataUpdater()
		store.table[oldKey] = fileMeta{
			Source:         path,
			IdentifierName: fingerprintName,
			Fingerprint:    oldFingerprint,
		}
		p := &fileProspector{
			logger:             log,
			identifier:         identifier,
			shortFingerprints:  newShortFingerprintSet(),
			growingFingerprint: true,
		}
		p.shortFingerprints.Add(oldKey, oldFingerprint, path)

		event := loginp.FSEvent{
			Op:         loginp.OpWrite,
			OldPath:    path,
			NewPath:    path,
			SrcID:      newKey,
			Descriptor: desc,
		}
		src := identifier.GetSource(event)

		hg := newTestHarvesterGroup()
		p.onFSEvent(log, input.Context{}, event, src, store, hg, time.Time{})

		assert.False(t, store.has(oldKey), "old key should have been removed by migration")
		assert.True(t, store.has(newKey), "new key should have been added by migration")
		assert.Len(t, store.table, 1, "registry should have exactly one entry")

		// The migrated entry is still growing: it persists the raw fingerprint
		// (the bounded-key marker for "still growing").
		gotMeta := store.table[newKey].(fileMeta)
		assert.Equal(t, newFingerprint, gotMeta.Fingerprint,
			"migrated entry should persist the raw growing fingerprint while below threshold")

		// Short fingerprint set tracks the new key, drops the old.
		assert.NotContains(t, p.shortFingerprints.entries, oldKey, "old entry removed from short fingerprint set")
		assert.Contains(t, p.shortFingerprints.entries, newKey, "new entry added to short fingerprint set")
		assert.Equal(t, newFingerprint, p.shortFingerprints.entries[newKey].Fingerprint)
	})

	t.Run("at threshold: raw-hex transitions to SHA-256", func(t *testing.T) {
		sha256Fingerprint := "1111111111111111111111111111111111111111111111111111111111111111"
		growingFingerprint := oldFingerprint + strings.Repeat("0", 60)
		newKey := "filestream::" + inputID + "::fingerprint::" + sha256Fingerprint

		store := newMockMetadataUpdater()
		store.table[oldKey] = fileMeta{
			Source:         path,
			IdentifierName: fingerprintName,
			Fingerprint:    oldFingerprint,
		}
		p := &fileProspector{
			logger:             log,
			identifier:         identifier,
			shortFingerprints:  newShortFingerprintSet(),
			growingFingerprint: true,
		}
		p.shortFingerprints.Add(oldKey, oldFingerprint, path)

		event := loginp.FSEvent{
			Op:      loginp.OpWrite,
			OldPath: path,
			NewPath: path,
			SrcID:   newKey,
			Descriptor: loginp.FileDescriptor{
				Fingerprint: loginp.FingerprintID{
					Sum: sha256Fingerprint,
					Raw: growingFingerprint,
				},
			},
		}
		src := identifier.GetSource(event)

		hg := newTestHarvesterGroup()
		p.onFSEvent(log, input.Context{}, event, src, store, hg, time.Time{})

		assert.False(t, store.has(oldKey), "old key should have been removed by migration")
		assert.True(t, store.has(newKey), "new key (SHA-256) should exist after migration")
		assert.Len(t, store.table, 1, "registry should have exactly one entry")

		// The migrated entry is final at threshold: the raw fingerprint is
		// cleared (omitted on disk → byte-identical to a static entry).
		gotMeta := store.table[newKey].(fileMeta)
		assert.Empty(t, gotMeta.Fingerprint,
			"migrated entry should clear the raw fingerprint at threshold")

		// Short fingerprint set drops the old (migrated away) and does NOT add
		// the new entry (it's final SHA-256, not growing anymore).
		assert.NotContains(t, p.shortFingerprints.entries, oldKey, "old entry removed from short fingerprint set")
		assert.NotContains(t, p.shortFingerprints.entries, newKey, "transitioned entry NOT added to short fingerprint set")
		assert.Empty(t, p.shortFingerprints.entries, "short fingerprint set is empty after transition")
	})
}

func TestBuildShortFingerprintSet(t *testing.T) {
	store := newMockMetadataUpdater()
	// Growing fingerprint entry (non-empty raw Fingerprint) — should be included
	store.table["filestream::input::fingerprint::aabb"] = fileMeta{
		Source:         "/a.log",
		IdentifierName: fingerprintName,
		Fingerprint:    "aabb",
	}
	// Final SHA-256 entry (empty Fingerprint) — should be excluded
	store.table["filestream::input::fingerprint::"+strings.Repeat("ab", 32)] = fileMeta{
		Source:         "/b.log",
		IdentifierName: fingerprintName,
	}
	// Legacy entry from a registry written before the feature existed
	// (no Fingerprint field present → empty on read → treated as final),
	// should be excluded
	store.table["filestream::input::fingerprint::ccddeeff"] = fileMeta{
		Source:         "/legacy.log",
		IdentifierName: fingerprintName,
	}
	// Empty raw fingerprint — treated as final, should be excluded
	store.table["filestream::input::fingerprint::"] = fileMeta{
		Source:         "/c.log",
		IdentifierName: fingerprintName,
	}
	// Non-fingerprint entry (e.g. native or path identity) — should be excluded
	store.table["filestream::input::native::abc123"] = fileMeta{
		Source:         "/d.log",
		IdentifierName: nativeName,
	}
	// Malformed key (too few separators) — should be excluded even though it
	// carries a raw fingerprint.
	store.table["filestream::malformed"] = fileMeta{
		Source:         "/e.log",
		IdentifierName: fingerprintName,
		Fingerprint:    "aabb",
	}

	p := &fileProspector{logger: logptest.NewTestingLogger(t, "")}
	p.buildShortFingerprintSet(store)

	require.Len(t, p.shortFingerprints.entries, 1)
	got, ok := p.shortFingerprints.entries["filestream::input::fingerprint::aabb"]
	require.True(t, ok, "expected the growing entry to be in the set")
	assert.Equal(t, "aabb", got.Fingerprint, "fingerprint mismatch")
	assert.Equal(t, "/a.log", got.Source, "source mismatch")
}

func TestShortFingerprintEntries_EventMaintenance(t *testing.T) {
	identifier, err := newFingerprintIdentifier(nil, nil)
	require.NoError(t, err)

	makeEvent := func(op loginp.Operation, path, srcID, fp string) loginp.FSEvent {
		return loginp.FSEvent{
			Op:      op,
			NewPath: path,
			OldPath: path,
			SrcID:   srcID,
			Descriptor: loginp.FileDescriptor{
				Fingerprint: loginp.FingerprintID{Raw: fp},
				Info:        file.ExtendFileInfo(&testFileInfo{path, 100, time.Now(), nil}),
			},
		}
	}

	t.Run("OpCreate with growing entry adds entry", func(t *testing.T) {
		p := &fileProspector{
			logger:            logptest.NewTestingLogger(t, ""),
			identifier:        identifier,
			shortFingerprints: newShortFingerprintSet(),
		}
		event := makeEvent(loginp.OpCreate, "/a.log", "filestream::input::fingerprint::aabb", "aabb")
		src := identifier.GetSource(event)
		store := newMockMetadataUpdater()
		hg := newTestHarvesterGroup()

		p.onFSEvent(logptest.NewTestingLogger(t, ""), input.Context{}, event, src, store, hg, time.Time{})

		require.Len(t, p.shortFingerprints.entries, 1)
		entry, ok := p.shortFingerprints.entries["filestream::input::fingerprint::aabb"]
		require.True(t, ok)
		assert.Equal(t, "aabb", entry.Fingerprint)
		assert.Equal(t, "/a.log", entry.Source)
	})

	t.Run("OpCreate with non-growing entry does NOT add entry", func(t *testing.T) {
		// Files already at threshold (completed SHA-256) are not added to the
		// short-fingerprint set: they cannot participate in prefix matching.
		const sha = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		p := &fileProspector{
			logger:            logptest.NewTestingLogger(t, ""),
			identifier:        identifier,
			shortFingerprints: newShortFingerprintSet(),
		}
		event := makeEvent(loginp.OpCreate, "/a.log", "filestream::input::fingerprint::"+sha, sha)
		event.Descriptor.Fingerprint = loginp.FingerprintID{Sum: sha}
		src := identifier.GetSource(event)
		store := newMockMetadataUpdater()
		hg := newTestHarvesterGroup()

		p.onFSEvent(logptest.NewTestingLogger(t, ""), input.Context{}, event, src, store, hg, time.Time{})

		assert.Empty(t, p.shortFingerprints.entries)
	})

	t.Run("OpDelete removes entry", func(t *testing.T) {
		srcID := "filestream::input::fingerprint::aabb"
		p := &fileProspector{
			logger:     logptest.NewTestingLogger(t, ""),
			identifier: identifier,
			shortFingerprints: &shortFingerprintSet{entries: map[string]shortFingerprintEntry{
				srcID: {Fingerprint: "aabb", Source: "/a.log"},
			}},
		}
		event := makeEvent(loginp.OpDelete, "/a.log", srcID, "aabb")
		event.OldPath = "/a.log"
		event.NewPath = ""
		src := identifier.GetSource(event)
		store := newMockMetadataUpdater()
		hg := newTestHarvesterGroup()

		p.onFSEvent(logptest.NewTestingLogger(t, ""), input.Context{}, event, src, store, hg, time.Time{})

		assert.Empty(t, p.shortFingerprints.entries)
	})

	t.Run("OpRename updates source path", func(t *testing.T) {
		srcID := "filestream::input::fingerprint::aabb"
		p := &fileProspector{
			logger:     logptest.NewTestingLogger(t, ""),
			identifier: identifier,
			shortFingerprints: &shortFingerprintSet{entries: map[string]shortFingerprintEntry{
				srcID: {Fingerprint: "aabb", Source: "/a.log"},
			}},
		}
		event := loginp.FSEvent{
			Op:      loginp.OpRename,
			OldPath: "/a.log",
			NewPath: "/a.log.1",
			SrcID:   srcID,
			Descriptor: loginp.FileDescriptor{
				Fingerprint: loginp.FingerprintID{Raw: "aabb"},
				Info:        file.ExtendFileInfo(&testFileInfo{"/a.log.1", 100, time.Now(), nil}),
			},
		}
		src := identifier.GetSource(event)
		store := newMockMetadataUpdater()
		hg := newTestHarvesterGroup()

		p.onFSEvent(logptest.NewTestingLogger(t, ""), input.Context{}, event, src, store, hg, time.Time{})

		require.Len(t, p.shortFingerprints.entries, 1)
		entry := p.shortFingerprints.entries[srcID]
		assert.Equal(t, "/a.log.1", entry.Source)
		assert.Equal(t, "aabb", entry.Fingerprint)
	})

	t.Run("OpTruncate removes stale entry by path", func(t *testing.T) {
		oldSrcID := "filestream::input::fingerprint::aabb"
		// After truncation, the SrcID is based on the NEW (truncated) fingerprint
		truncatedSrcID := "filestream::input::fingerprint::xx"
		p := &fileProspector{
			logger:     logptest.NewTestingLogger(t, ""),
			identifier: identifier,
			shortFingerprints: &shortFingerprintSet{entries: map[string]shortFingerprintEntry{
				oldSrcID: {Fingerprint: "aabb", Source: "/a.log"},
			}},
		}
		event := makeEvent(loginp.OpTruncate, "/a.log", truncatedSrcID, "xx")
		src := identifier.GetSource(event)
		store := newMockMetadataUpdater()
		hg := newTestHarvesterGroup()

		p.onFSEvent(logptest.NewTestingLogger(t, ""), input.Context{}, event, src, store, hg, time.Time{})

		assert.Empty(t, p.shortFingerprints.entries,
			"stale entry should be removed by path match")
	})
}

func TestShortFingerprintEntries_MigrationMaintenance(t *testing.T) {
	identifier, err := newFingerprintIdentifier(nil, nil)
	require.NoError(t, err)

	t.Run("growing to growing: old removed, new added", func(t *testing.T) {
		oldFingerprint := "aabb"     // 4 chars, growing
		newFingerprint := "aabbccdd" // 8 chars, still growing
		path := "/a.log"
		oldKey := "filestream::input::fingerprint::" + oldFingerprint
		newSrcID := "filestream::input::fingerprint::" + newFingerprint

		store := newMockMetadataUpdater()
		store.table[oldKey] = fileMeta{Source: path, IdentifierName: fingerprintName, Fingerprint: oldFingerprint}

		p := &fileProspector{
			logger:     logptest.NewTestingLogger(t, ""),
			identifier: identifier,
			shortFingerprints: &shortFingerprintSet{entries: map[string]shortFingerprintEntry{
				oldKey: {Fingerprint: oldFingerprint, Source: path},
			}},
		}

		event := loginp.FSEvent{
			NewPath: path,
			SrcID:   newSrcID,
			Descriptor: loginp.FileDescriptor{
				Fingerprint: loginp.FingerprintID{Raw: newFingerprint},
				Info:        file.ExtendFileInfo(&testFileInfo{path, 100, time.Now(), nil}),
			},
		}
		src := identifier.GetSource(event)

		p.handleGrowingFingerprintLookup(logptest.NewTestingLogger(t, ""), event, src, store)

		assert.NotContains(t, p.shortFingerprints.entries, oldKey,
			"old entry should be removed")
		require.Contains(t, p.shortFingerprints.entries, newSrcID,
			"new entry should be added")
		assert.Equal(t, newFingerprint, p.shortFingerprints.entries[newSrcID].Fingerprint)
		assert.Equal(t, path, p.shortFingerprints.entries[newSrcID].Source)
	})

	t.Run("growing to threshold (completed): old removed, new NOT added", func(t *testing.T) {
		// File transitioned from raw-hex to SHA-256 at threshold. The
		// descriptor still carries the raw header in Raw so the existing short
		// fingerprint entry can be located; after migration the new entry is
		// final and must not be re-added to the short fingerprint set.
		oldFingerprint := "aabb"
		path := "/a.log"
		newFingerprint := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
		growingFingerprint := "aabb" + strings.Repeat("0", 60)
		oldKey := "filestream::input::fingerprint::" + oldFingerprint
		newSrcID := "filestream::input::fingerprint::" + newFingerprint

		store := newMockMetadataUpdater()
		store.table[oldKey] = fileMeta{Source: path, IdentifierName: fingerprintName, Fingerprint: oldFingerprint}

		p := &fileProspector{
			logger:     logptest.NewTestingLogger(t, ""),
			identifier: identifier,
			shortFingerprints: &shortFingerprintSet{entries: map[string]shortFingerprintEntry{
				oldKey: {Fingerprint: oldFingerprint, Source: path},
			}},
		}

		event := loginp.FSEvent{
			NewPath: path,
			SrcID:   newSrcID,
			Descriptor: loginp.FileDescriptor{
				Fingerprint: loginp.FingerprintID{
					Sum: newFingerprint,
					Raw: growingFingerprint,
				},
				Info: file.ExtendFileInfo(&testFileInfo{path, 100, time.Now(), nil}),
			},
		}
		src := identifier.GetSource(event)

		p.handleGrowingFingerprintLookup(logptest.NewTestingLogger(t, ""), event, src, store)

		assert.NotContains(t, p.shortFingerprints.entries, oldKey, "old entry should be removed")
		assert.NotContains(t, p.shortFingerprints.entries, newSrcID, "transitioned entry should NOT be added")
		assert.Empty(t, p.shortFingerprints.entries)
	})

	t.Run("input ID containing 'fingerprint' substring: migration succeeds", func(t *testing.T) {
		inputID := "test-fingerprint"
		oldFingerprint := "aabb"
		newFingerprint := "aabbccdd"
		path := "/a.log"
		oldKey := "filestream::" + inputID + "::fingerprint::" + oldFingerprint
		// The migrated key uses the bounded (hashed) fingerprint, not the raw fp.
		newSrcID := "filestream::" + inputID + "::fingerprint::" +
			loginp.FingerprintID{Raw: newFingerprint}.Key()

		store := newMockMetadataUpdater()
		store.table[oldKey] = fileMeta{
			Source:         path,
			IdentifierName: fingerprintName,
			Fingerprint:    oldFingerprint,
		}

		p := &fileProspector{
			logger:     logptest.NewTestingLogger(t, ""),
			identifier: identifier,
			shortFingerprints: &shortFingerprintSet{
				entries: map[string]shortFingerprintEntry{
					oldKey: {Fingerprint: oldFingerprint, Source: path},
				}},
		}

		event := loginp.FSEvent{
			NewPath: path,
			SrcID:   newSrcID,
			Descriptor: loginp.FileDescriptor{
				Fingerprint: loginp.FingerprintID{Raw: newFingerprint},
				Info: file.ExtendFileInfo(
					&testFileInfo{path, 100, time.Now(), nil}),
			},
		}
		src := identifier.GetSource(event)

		p.handleGrowingFingerprintLookup(logptest.NewTestingLogger(t, ""), event, src, store)

		assert.False(t, store.has(oldKey), "old key should be removed by migration")
		assert.True(t, store.has(newSrcID), "new key should exist after migration")
		assert.NotContains(t, p.shortFingerprints.entries, oldKey,
			"old entry should be removed from short fingerprint set")
		require.Contains(t, p.shortFingerprints.entries, newSrcID,
			"new entry should be added to short fingerprint set")
	})
}

func TestShortFingerprintEntries_FullLifecycle(t *testing.T) {
	log := logptest.NewTestingLogger(t, "")
	identifier, err := newFingerprintIdentifier(nil, nil)
	require.NoError(t, err)

	// makeKey returns the registry key the scanner/identifier would produce for
	// a descriptor: the bounded (hashed) key for a still-growing fingerprint, or
	// the SHA-256 as-is for a final one. This keeps the test's keys consistent
	// with what the migration code computes via FingerprintID.Key.
	makeKey := func(fingerprint string, growing bool) string {
		fp := loginp.FingerprintID{Sum: fingerprint}
		if growing {
			fp = loginp.FingerprintID{Raw: fingerprint}
		}
		return "filestream::input::fingerprint::" + fp.Key()
	}

	p := &fileProspector{
		logger:             log,
		identifier:         identifier,
		shortFingerprints:  newShortFingerprintSet(),
		growingFingerprint: true,
	}
	store := newMockMetadataUpdater()
	hg := newTestHarvesterGroup()

	// An entry is "still growing" while its raw Fingerprint is non-empty; the
	// final SHA-256 transition clears it.
	//   file A: raw-hex "aa" -> raw-hex "aabb" -> SHA-256 (final)
	//   file B: raw-hex "bb" -> SHA-256 (final)
	//   file C: raw-hex "cc" -> deleted

	// --- Cycle 1: Create 3 growing files ---
	initialFingerprints := map[string]string{
		"/a.log": "aa",
		"/b.log": "bb",
		"/c.log": "cc",
	}
	for _, path := range []string{"/a.log", "/b.log", "/c.log"} {
		fingerprint := initialFingerprints[path]
		event := loginp.FSEvent{
			Op:      loginp.OpCreate,
			NewPath: path,
			SrcID:   makeKey(fingerprint, true),
			Descriptor: loginp.FileDescriptor{
				Fingerprint: loginp.FingerprintID{Raw: fingerprint},
				Info:        file.ExtendFileInfo(&testFileInfo{path, 100, time.Now(), nil}),
			},
		}
		src := identifier.GetSource(event)
		store.table[makeKey(fingerprint, true)] = fileMeta{Source: path, IdentifierName: fingerprintName, Fingerprint: fingerprint}
		p.onFSEvent(log, input.Context{}, event, src, store, hg, time.Time{})
	}
	assert.Equal(t, map[string]shortFingerprintEntry{
		makeKey("aa", true): {Fingerprint: "aa", Source: "/a.log"},
		makeKey("bb", true): {Fingerprint: "bb", Source: "/b.log"},
		makeKey("cc", true): {Fingerprint: "cc", Source: "/c.log"},
	}, p.shortFingerprints.entries, "cycle 1: all 3 growing entries present")

	// --- Cycle 2: file A still growing: "aa" -> "aabb" ---
	event := loginp.FSEvent{
		Op:      loginp.OpWrite,
		NewPath: "/a.log",
		SrcID:   makeKey("aabb", true),
		Descriptor: loginp.FileDescriptor{
			Fingerprint: loginp.FingerprintID{Raw: "aabb"},
			Info:        file.ExtendFileInfo(&testFileInfo{"/a.log", 200, time.Now(), nil}),
		},
	}
	src := identifier.GetSource(event)
	p.onFSEvent(log, input.Context{}, event, src, store, hg, time.Time{})
	assert.Equal(t, map[string]shortFingerprintEntry{
		makeKey("aabb", true): {Fingerprint: "aabb", Source: "/a.log"},
		makeKey("bb", true):   {Fingerprint: "bb", Source: "/b.log"}, // unchanged
		makeKey("cc", true):   {Fingerprint: "cc", Source: "/c.log"}, // unchanged
	}, p.shortFingerprints.entries, "cycle 2: file A migrated aa->aabb, B and C unchanged")

	// --- Cycle 3: file A reaches threshold: raw-hex "aabb" -> SHA-256 ---
	aSha := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	event = loginp.FSEvent{
		Op:      loginp.OpWrite,
		NewPath: "/a.log",
		SrcID:   makeKey(aSha, false),
		Descriptor: loginp.FileDescriptor{
			Fingerprint: loginp.FingerprintID{
				Sum: aSha,
				Raw: "aabb" + strings.Repeat("0", 60), // raw-hex of bytes[offset:offset+length]
			},
			Info: file.ExtendFileInfo(&testFileInfo{"/a.log", 500, time.Now(), nil}),
		},
	}
	src = identifier.GetSource(event)
	p.onFSEvent(logptest.NewTestingLogger(t, ""), input.Context{}, event, src, store, hg, time.Time{})
	assert.Equal(t, map[string]shortFingerprintEntry{
		makeKey("bb", true): {Fingerprint: "bb", Source: "/b.log"},
		makeKey("cc", true): {Fingerprint: "cc", Source: "/c.log"},
	}, p.shortFingerprints.entries, "cycle 3: file A transitioned to SHA-256, removed from set")

	// --- Cycle 4: file B reaches threshold: raw-hex "bb" -> SHA-256 ---
	const bSha = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	event = loginp.FSEvent{
		Op:      loginp.OpWrite,
		NewPath: "/b.log",
		SrcID:   makeKey(bSha, false),
		Descriptor: loginp.FileDescriptor{
			Fingerprint: loginp.FingerprintID{
				Sum: bSha,
				Raw: "bb" + strings.Repeat("0", 62),
			},
			Info: file.ExtendFileInfo(&testFileInfo{"/b.log", 500, time.Now(), nil}),
		},
	}
	src = identifier.GetSource(event)
	p.onFSEvent(log, input.Context{}, event, src, store, hg, time.Time{})
	assert.Equal(t, map[string]shortFingerprintEntry{
		makeKey("cc", true): {Fingerprint: "cc", Source: "/c.log"},
	}, p.shortFingerprints.entries, "cycle 4: file B transitioned to SHA-256, only C remains")

	// --- Cycle 5: file C deleted ---
	event = loginp.FSEvent{
		Op:      loginp.OpDelete,
		OldPath: "/c.log",
		SrcID:   makeKey("cc", true),
		Descriptor: loginp.FileDescriptor{
			Fingerprint: loginp.FingerprintID{Raw: "cc"},
			Info:        file.ExtendFileInfo(&testFileInfo{"/c.log", 100, time.Now(), nil}),
		},
	}
	src = identifier.GetSource(event)
	p.onFSEvent(log, input.Context{}, event, src, store, hg, time.Time{})
	assert.Empty(t, p.shortFingerprints.entries, "cycle 5: file C deleted, set is empty")
}

func mustInodeMarker(t *testing.T) fileIdentifier {
	f, err := os.CreateTemp(t.TempDir(), "inode-marker")
	if err != nil {
		t.Fatalf("cannot create inode marker: %s", err)
	}

	fullPath, err := filepath.Abs(f.Name())
	if err != nil {
		t.Fatalf("cannot get full path from file: %s", err)
	}

	if _, err := fmt.Fprint(f, "foo-bar"); err != nil {
		t.Fatalf("cannot write to inode-marker file: %s", err)
	}

	if err := f.Sync(); err != nil {
		t.Fatalf("cannot sync file: %s", err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("cannot close file: %s", err)
	}

	cfg := conf.MustNewConfigFrom("path: " + fullPath)
	identifier, err := newINodeMarkerIdentifier(cfg, logp.NewNopLogger())
	if err != nil {
		t.Fatalf("cannot create inode marker identifier: %s", err)
	}
	return identifier
}
