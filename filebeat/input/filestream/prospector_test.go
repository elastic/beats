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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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
		testCase := testCase

		t.Run(name, func(t *testing.T) {
			testStore := newMockStoreUpdater(testCase.entries)
			p := fileProspector{
				logger:       logp.L(),
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
		testCase := testCase

		t.Run(name, func(t *testing.T) {
			testStore := newMockStoreUpdater(testCase.entries)
			p := fileProspector{
				logger:      logp.L(),
				identifier:  mustPathIdentifier(false),
				filewatcher: newMockFileWatcherWithFiles(testCase.filesOnDisk),
			}
			err := p.Init(testStore, newMockStoreUpdater(nil), func(loginp.Source) string { return testCase.newKey })
			require.NoError(t, err, "prospector Init must succeed")
			assert.Equal(t, testCase.expectedUpdatedKeys, testStore.updatedKeys)
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
		Fingerprint: mockFingerprint,
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
				logger:      logp.L(),
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
		test := test

		t.Run(name, func(t *testing.T) {
			p := fileProspector{
				logger:      logp.L(),
				filewatcher: newMockFileWatcher(test.events, len(test.events)),
				identifier:  mustPathIdentifier(false),
				ignoreOlder: test.ignoreOlder,
			}
			ctx := input.Context{Logger: logp.L(), Cancelation: context.Background()}
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
		logger:      logp.L(),
		filewatcher: filewatcher,
		identifier:  mustPathIdentifier(false),
		ignoreOlder: 10 * time.Second,
	}
	ctx := input.Context{Logger: logp.L(), Cancelation: context.Background()}
	hg := newTestHarvesterGroup()
	testStore := newMockMetadataUpdater()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		p.Run(ctx, testStore, hg)

		wg.Done()
	}()

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
		test := test

		t.Run(name, func(t *testing.T) {
			p := fileProspector{
				logger:       logp.L(),
				filewatcher:  newMockFileWatcher(test.events, len(test.events)),
				identifier:   mustPathIdentifier(false),
				cleanRemoved: test.cleanRemoved,
			}
			ctx := input.Context{Logger: logp.L(), Cancelation: context.Background()}

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
		test := test

		t.Run(name, func(t *testing.T) {
			p := fileProspector{
				logger:            logp.L(),
				filewatcher:       newMockFileWatcher(test.events, len(test.events)),
				identifier:        mustPathIdentifier(test.trackRename),
				stateChangeCloser: stateChangeCloserConfig{Renamed: test.closeRenamed},
			}
			ctx := input.Context{Logger: logp.L(), Cancelation: context.Background()}

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

type mockMetadataUpdater struct {
	table map[string]interface{}

	FindCursorMetaCalled  int
	ResetCursorCalled     int
	UpdateMetadataCalled  int
	RemoveCalled          int
	IterateOnPrefixCalled int
	KeyExistsCalled       int
	UpdateKeyCalled       int
}

func newMockMetadataUpdater() *mockMetadataUpdater {
	return &mockMetadataUpdater{
		table: make(map[string]interface{}),
	}
}

func (mu *mockMetadataUpdater) set(id string) { mu.table[id] = struct{}{} }

func (mu *mockMetadataUpdater) has(id string) bool {
	_, ok := mu.table[id]
	return ok
}

func (mu *mockMetadataUpdater) checkOffset(id string, offset int64) bool {
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

func (mu *mockMetadataUpdater) FindCursorMeta(s loginp.Source, v interface{}) error {
	mu.FindCursorMetaCalled++
	meta, ok := mu.table[s.Name()]
	if !ok {
		return fmt.Errorf("no such id [%q]", s.Name())
	}
	return typeconv.Convert(v, meta)
}

func (mu *mockMetadataUpdater) ResetCursor(s loginp.Source, cur interface{}) error {
	mu.ResetCursorCalled++
	mu.table[s.Name()] = cur
	return nil
}

func (mu *mockMetadataUpdater) UpdateMetadata(s loginp.Source, v interface{}) error {
	mu.UpdateMetadataCalled++
	mu.table[s.Name()] = v
	return nil
}

func (mu *mockMetadataUpdater) Remove(s loginp.Source) error {
	mu.RemoveCalled++
	delete(mu.table, s.Name())
	return nil
}

func (mu *mockMetadataUpdater) IterateOnPrefix(fn func(key string, meta interface{}) bool) {
	mu.IterateOnPrefixCalled++
	for key, meta := range mu.table {
		if !fn(key, meta) {
			return
		}
	}
}

func (mu *mockMetadataUpdater) KeyExists(key string) bool {
	mu.KeyExistsCalled++
	_, ok := mu.table[key]
	return ok
}

func (mu *mockMetadataUpdater) UpdateKey(oldKey, newKey string, meta interface{}) error {
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

func (u *mockUnpackValue) UnpackCursorMeta(to interface{}) error {
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
				testStore.table[id] = fileMeta{Source: path, IdentifierName: expectedIdentifier}
			}

			hg := newTestHarvesterGroup()
			p.Run(ctx, testStore, hg)

			got := testStore.table[id]
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
	sys  interface{}
}

func (t *testFileInfo) Name() string       { return t.name }
func (t *testFileInfo) Size() int64        { return t.size }
func (t *testFileInfo) Mode() os.FileMode  { return 0 }
func (t *testFileInfo) ModTime() time.Time { return t.time }
func (t *testFileInfo) IsDir() bool        { return false }
func (t *testFileInfo) Sys() interface{}   { return t.sys }

func createTestFileDescriptor() loginp.FileDescriptor {
	return createTestFileDescriptorWithInfo(&testFileInfo{})
}

func createTestFileDescriptorWithInfo(fi fs.FileInfo) loginp.FileDescriptor {
	return loginp.FileDescriptor{
		Info:        file.ExtendFileInfo(fi),
		Fingerprint: "fingerprint",
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
		Fingerprint: "test-fingerprint",
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
		Fingerprint: "test-fingerprint",
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
		storeEntries       map[string]interface{}
		currentFingerprint string
		currentPath        string
		expectedKey        string
		expectedFound      bool
	}{
		"empty current fingerprint returns immediately": {
			storeEntries:       map[string]interface{}{},
			currentFingerprint: "",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"valid prefix match": {
			storeEntries: map[string]interface{}{
				"filestream::my-input::growing_fingerprint::aabb": fileMeta{
					Source:         currentPath,
					IdentifierName: growingFingerprintName,
				},
			},
			currentFingerprint: "aabbccdd",
			currentPath:        currentPath,
			expectedKey:        "filestream::my-input::growing_fingerprint::aabb",
			expectedFound:      true,
		},
		"prefix match among entries for different paths": {
			storeEntries: map[string]interface{}{
				"filestream::my-input::growing_fingerprint::aa": fileMeta{
					Source:         "/other/file.log",
					IdentifierName: growingFingerprintName,
				},
				"filestream::my-input::growing_fingerprint::aabb": fileMeta{
					Source:         currentPath,
					IdentifierName: growingFingerprintName,
				},
			},
			currentFingerprint: "aabbccddee",
			currentPath:        currentPath,
			expectedKey:        "filestream::my-input::growing_fingerprint::aabb",
			expectedFound:      true,
		},
		"skips non-growing_fingerprint identity": {
			storeEntries: map[string]interface{}{
				"filestream::my-input::fingerprint::aabb": fileMeta{
					Source:         currentPath,
					IdentifierName: fingerprintName,
				},
			},
			currentFingerprint: "aabbccdd",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"skips key with too many separators": {
			storeEntries: map[string]interface{}{
				"filestream::my-input::growing_fingerprint::aabb::extra": fileMeta{
					Source:         currentPath,
					IdentifierName: growingFingerprintName,
				},
			},
			currentFingerprint: "aabbccdd",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"skips key with too few separators": {
			storeEntries: map[string]interface{}{
				"filestream::growing_fingerprint": fileMeta{
					Source:         currentPath,
					IdentifierName: growingFingerprintName,
				},
			},
			currentFingerprint: "aabbccdd",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"skips empty stored fingerprint": {
			storeEntries: map[string]interface{}{
				"filestream::my-input::growing_fingerprint::": fileMeta{
					Source:         currentPath,
					IdentifierName: growingFingerprintName,
				},
			},
			currentFingerprint: "aabbccdd",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"skips stored fingerprint longer than current": {
			storeEntries: map[string]interface{}{
				"filestream::my-input::growing_fingerprint::aabbccddee": fileMeta{
					Source:         currentPath,
					IdentifierName: growingFingerprintName,
				},
			},
			currentFingerprint: "aabb",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"skips stored fingerprint equal length to current": {
			storeEntries: map[string]interface{}{
				"filestream::my-input::growing_fingerprint::aabb": fileMeta{
					Source:         currentPath,
					IdentifierName: growingFingerprintName,
				},
			},
			currentFingerprint: "aabb",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"skips non-prefix fingerprint": {
			storeEntries: map[string]interface{}{
				"filestream::my-input::growing_fingerprint::xxxx": fileMeta{
					Source:         currentPath,
					IdentifierName: growingFingerprintName,
				},
			},
			currentFingerprint: "aabbccdd",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"skips path mismatch": {
			storeEntries: map[string]interface{}{
				"filestream::my-input::growing_fingerprint::aabb": fileMeta{
					Source:         "/other/file.log",
					IdentifierName: growingFingerprintName,
				},
			},
			currentFingerprint: "aabbccdd",
			currentPath:        currentPath,
			expectedFound:      false,
		},
		"single colon in input ID is not a separator": {
			storeEntries: map[string]interface{}{
				"filestream::my:input::growing_fingerprint::aabb": fileMeta{
					Source:         currentPath,
					IdentifierName: growingFingerprintName,
				},
			},
			currentFingerprint: "aabbccdd",
			currentPath:        currentPath,
			expectedKey:        "filestream::my:input::growing_fingerprint::aabb",
			expectedFound:      true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			store := newMockMetadataUpdater()
			for k, v := range tc.storeEntries {
				store.table[k] = v
			}

			p := &fileProspector{logger: logp.L(), maxEncodedFingerprintLen: 2000}
			key, found := p.findGrowingFingerprintMatch(store, tc.currentFingerprint, tc.currentPath)

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

	identifier, err := newGrowingFingerprintIdentifier(nil, nil)
	require.NoError(t, err)

	event := loginp.FSEvent{
		NewPath: currentPath,
		SrcID:   "filestream::my-input::growing_fingerprint::" + currentFingerprint,
		Descriptor: loginp.FileDescriptor{
			Fingerprint: currentFingerprint,
		},
	}

	src := identifier.GetSource(event)

	t.Run("fast path: key exists skips scan", func(t *testing.T) {
		store := newMockMetadataUpdater()
		// The store already has an entry for the exact current fingerprint key.
		store.table[event.SrcID] = fileMeta{
			Source:         currentPath,
			IdentifierName: growingFingerprintName,
		}

		// Also add an entry that would be a prefix match (the slow path).
		// If the fast path works, it should never reach findGrowingFingerprintMatch.
		store.table["filestream::my-input::growing_fingerprint::aabb"] = fileMeta{
			Source:         currentPath,
			IdentifierName: growingFingerprintName,
		}

		p := &fileProspector{
			logger:                   logp.L(),
			identifier:               identifier,
			maxEncodedFingerprintLen: 2000,
		}

		result := p.handleGrowingFingerprintLookup(logp.L(), event, src, store)
		assert.Equal(t, src.Name(), result.Name(), "fast path should return original src unchanged")
	})

	t.Run("slow path: key does not exist falls through to scan", func(t *testing.T) {
		store := newMockMetadataUpdater()
		// Only a prefix match exists, not the exact key.
		store.table["filestream::my-input::growing_fingerprint::aabb"] = fileMeta{
			Source:         currentPath,
			IdentifierName: growingFingerprintName,
		}

		p := &fileProspector{
			logger:                   logp.L(),
			identifier:               identifier,
			maxEncodedFingerprintLen: 2000,
		}

		result := p.handleGrowingFingerprintLookup(logp.L(), event, src, store)
		// The function returns src (it always does). Migration succeeds via
		// the short fingerprint set prefix match.
		assert.Equal(t, src.Name(), result.Name())
	})
}

// TestOnFSEvent_GrowingFingerprintMaxLen verifies that maxEncodedFingerprintLen
// controls whether handleGrowingFingerprintLookup is called inside onFSEvent.
func TestOnFSEvent_GrowingFingerprintMaxLen(t *testing.T) {
	const (
		currentPath    = "/var/log/app.log"
		oldFingerprint = "aabb"
		newFingerprint = "aabbccdd"
		inputID        = "my-input"
	)

	oldKey := "filestream::" + inputID + "::growing_fingerprint::" + oldFingerprint
	newKey := "filestream::" + inputID + "::growing_fingerprint::" + newFingerprint

	identifier, err := newGrowingFingerprintIdentifier(nil, nil)
	require.NoError(t, err)

	event := loginp.FSEvent{
		Op:      loginp.OpWrite,
		NewPath: currentPath,
		SrcID:   newKey,
		Descriptor: loginp.FileDescriptor{
			Fingerprint: newFingerprint,
		},
	}
	src := identifier.GetSource(event)

	t.Run("below max: fingerprint grew, migration happens", func(t *testing.T) {
		store := newMockMetadataUpdater()
		store.table[oldKey] = fileMeta{
			Source:         currentPath,
			IdentifierName: growingFingerprintName,
		}

		p := &fileProspector{
			logger:                   logp.L(),
			identifier:               identifier,
			maxEncodedFingerprintLen: len(newFingerprint) + 1,
		}

		hg := newTestHarvesterGroup()
		p.onFSEvent(logp.L(), input.Context{}, event, src, store, hg, time.Time{})

		assert.False(t, store.has(oldKey), "old key should have been removed by migration")
		assert.True(t, store.has(newKey), "new key should exist after migration")
		assert.Len(t, store.table, 1, "it should have exactly one entry")
	})

	t.Run("at max: fingerprint reached max, migration still happens", func(t *testing.T) {
		store := newMockMetadataUpdater()
		store.table[oldKey] = fileMeta{
			Source:         currentPath,
			IdentifierName: growingFingerprintName,
		}

		p := &fileProspector{
			logger:                   logp.L(),
			identifier:               identifier,
			maxEncodedFingerprintLen: len(newFingerprint),
		}

		hg := newTestHarvesterGroup()
		p.onFSEvent(logp.L(), input.Context{}, event, src, store, hg, time.Time{})

		assert.False(t, store.has(oldKey), "old key should have been removed by migration")
		assert.True(t, store.has(newKey), "new key should exist after migration")
		assert.Len(t, store.table, 1, "it should have exactly one entry")
	})
}

func TestBuildShortFingerprintSet(t *testing.T) {
	const maxEncLen = 100 // 50-byte fingerprint = 100 hex chars

	store := newMockMetadataUpdater()
	// Short growing_fingerprint entry — should be included
	store.table["filestream::input::growing_fingerprint::aabb"] = fileMeta{
		Source:         "/a.log",
		IdentifierName: growingFingerprintName,
	}
	// Max-length growing_fingerprint entry — should be excluded
	store.table["filestream::input::growing_fingerprint::"+strings.Repeat("ab", 50)] = fileMeta{
		Source:         "/b.log",
		IdentifierName: growingFingerprintName,
	}
	// Empty fingerprint — should be excluded
	store.table["filestream::input::growing_fingerprint::"] = fileMeta{
		Source:         "/c.log",
		IdentifierName: growingFingerprintName,
	}
	// Non-growing_fingerprint entry — should be excluded
	store.table["filestream::input::fingerprint::aabb"] = fileMeta{
		Source:         "/d.log",
		IdentifierName: fingerprintName,
	}
	// Malformed key (too few separators) — should be excluded
	store.table["filestream::growing_fingerprint"] = fileMeta{
		Source:         "/e.log",
		IdentifierName: growingFingerprintName,
	}

	p := &fileProspector{
		logger:                   logp.L(),
		maxEncodedFingerprintLen: maxEncLen,
	}
	p.buildShortFingerprintSet(store)

	require.Len(t, p.shortFingerprintIdx.entries, 1)
	entry, ok := p.shortFingerprintIdx.entries["filestream::input::growing_fingerprint::aabb"]
	require.True(t, ok, "expected short entry to be in set")
	assert.Equal(t, "aabb", entry.Fingerprint)
	assert.Equal(t, "/a.log", entry.Source)
}

func TestShortFingerprintEntries_EventMaintenance(t *testing.T) {
	const maxEncLen = 100

	identifier, err := newGrowingFingerprintIdentifier(nil, nil)
	require.NoError(t, err)

	makeEvent := func(op loginp.Operation, path, srcID, fp string) loginp.FSEvent {
		return loginp.FSEvent{
			Op:      op,
			NewPath: path,
			OldPath: path,
			SrcID:   srcID,
			Descriptor: loginp.FileDescriptor{
				Fingerprint: fp,
				Info:        file.ExtendFileInfo(&testFileInfo{path, 100, time.Now(), nil}),
			},
		}
	}

	t.Run("OpCreate with short fingerprint adds entry", func(t *testing.T) {
		p := &fileProspector{
			logger:                   logp.L(),
			identifier:               identifier,
			maxEncodedFingerprintLen: maxEncLen,
			shortFingerprintIdx:      newShortFingerprintIndex(maxEncLen),
		}
		event := makeEvent(loginp.OpCreate, "/a.log", "filestream::input::growing_fingerprint::aabb", "aabb")
		src := identifier.GetSource(event)
		store := newMockMetadataUpdater()
		hg := newTestHarvesterGroup()

		p.onFSEvent(logp.L(), input.Context{}, event, src, store, hg, time.Time{})

		require.Len(t, p.shortFingerprintIdx.entries, 1)
		entry, ok := p.shortFingerprintIdx.entries["filestream::input::growing_fingerprint::aabb"]
		require.True(t, ok)
		assert.Equal(t, "aabb", entry.Fingerprint)
		assert.Equal(t, "/a.log", entry.Source)
	})

	t.Run("OpCreate with max-length fingerprint does NOT add entry", func(t *testing.T) {
		maxFP := strings.Repeat("ab", 50) // 100 chars = maxEncLen
		p := &fileProspector{
			logger:                   logp.L(),
			identifier:               identifier,
			maxEncodedFingerprintLen: maxEncLen,
			shortFingerprintIdx:      newShortFingerprintIndex(maxEncLen),
		}
		event := makeEvent(loginp.OpCreate, "/a.log", "filestream::input::growing_fingerprint::"+maxFP, maxFP)
		src := identifier.GetSource(event)
		store := newMockMetadataUpdater()
		hg := newTestHarvesterGroup()

		p.onFSEvent(logp.L(), input.Context{}, event, src, store, hg, time.Time{})

		assert.Len(t, p.shortFingerprintIdx.entries, 0)
	})

	t.Run("OpDelete removes entry", func(t *testing.T) {
		srcID := "filestream::input::growing_fingerprint::aabb"
		p := &fileProspector{
			logger:                   logp.L(),
			identifier:               identifier,
			maxEncodedFingerprintLen: maxEncLen,
			shortFingerprintIdx: &shortFingerprintIndex{maxLen: maxEncLen, entries: map[string]shortFingerprintEntry{
				srcID: {Fingerprint: "aabb", Source: "/a.log"},
			}},
		}
		event := makeEvent(loginp.OpDelete, "/a.log", srcID, "aabb")
		event.OldPath = "/a.log"
		event.NewPath = ""
		src := identifier.GetSource(event)
		store := newMockMetadataUpdater()
		hg := newTestHarvesterGroup()

		p.onFSEvent(logp.L(), input.Context{}, event, src, store, hg, time.Time{})

		assert.Len(t, p.shortFingerprintIdx.entries, 0)
	})

	t.Run("OpRename updates source path", func(t *testing.T) {
		srcID := "filestream::input::growing_fingerprint::aabb"
		p := &fileProspector{
			logger:                   logp.L(),
			identifier:               identifier,
			maxEncodedFingerprintLen: maxEncLen,
			shortFingerprintIdx: &shortFingerprintIndex{maxLen: maxEncLen, entries: map[string]shortFingerprintEntry{
				srcID: {Fingerprint: "aabb", Source: "/a.log"},
			}},
		}
		event := loginp.FSEvent{
			Op:      loginp.OpRename,
			OldPath: "/a.log",
			NewPath: "/a.log.1",
			SrcID:   srcID,
			Descriptor: loginp.FileDescriptor{
				Fingerprint: "aabb",
				Info:        file.ExtendFileInfo(&testFileInfo{"/a.log.1", 100, time.Now(), nil}),
			},
		}
		src := identifier.GetSource(event)
		store := newMockMetadataUpdater()
		hg := newTestHarvesterGroup()

		p.onFSEvent(logp.L(), input.Context{}, event, src, store, hg, time.Time{})

		require.Len(t, p.shortFingerprintIdx.entries, 1)
		entry := p.shortFingerprintIdx.entries[srcID]
		assert.Equal(t, "/a.log.1", entry.Source)
		assert.Equal(t, "aabb", entry.Fingerprint)
	})

	t.Run("OpTruncate removes stale entry by path", func(t *testing.T) {
		oldSrcID := "filestream::input::growing_fingerprint::aabb"
		// After truncation, the SrcID is based on the NEW (truncated) fingerprint
		truncatedSrcID := "filestream::input::growing_fingerprint::xx"
		p := &fileProspector{
			logger:                   logp.L(),
			identifier:               identifier,
			maxEncodedFingerprintLen: maxEncLen,
			shortFingerprintIdx: &shortFingerprintIndex{maxLen: maxEncLen, entries: map[string]shortFingerprintEntry{
				oldSrcID: {Fingerprint: "aabb", Source: "/a.log"},
			}},
		}
		event := makeEvent(loginp.OpTruncate, "/a.log", truncatedSrcID, "xx")
		src := identifier.GetSource(event)
		store := newMockMetadataUpdater()
		hg := newTestHarvesterGroup()

		p.onFSEvent(logp.L(), input.Context{}, event, src, store, hg, time.Time{})

		assert.Len(t, p.shortFingerprintIdx.entries, 0, "stale entry should be removed by path match")
	})
}

func TestShortFingerprintEntries_MigrationMaintenance(t *testing.T) {
	const maxEncLen = 20 // fingerprints >= 20 chars are "at max"

	identifier, err := newGrowingFingerprintIdentifier(nil, nil)
	require.NoError(t, err)

	t.Run("short to short: old removed, new added", func(t *testing.T) {
		const (
			oldFP = "aabb"     // 4 chars, short
			newFP = "aabbccdd" // 8 chars, still short (< 20)
			path  = "/a.log"
		)
		oldKey := "filestream::input::growing_fingerprint::" + oldFP
		newSrcID := "filestream::input::growing_fingerprint::" + newFP

		store := newMockMetadataUpdater()
		store.table[oldKey] = fileMeta{Source: path, IdentifierName: growingFingerprintName}

		p := &fileProspector{
			logger:                   logp.L(),
			identifier:               identifier,
			maxEncodedFingerprintLen: maxEncLen,
			shortFingerprintIdx: &shortFingerprintIndex{maxLen: maxEncLen, entries: map[string]shortFingerprintEntry{
				oldKey: {Fingerprint: oldFP, Source: path},
			}},
		}

		event := loginp.FSEvent{
			NewPath: path,
			SrcID:   newSrcID,
			Descriptor: loginp.FileDescriptor{
				Fingerprint: newFP,
				Info:        file.ExtendFileInfo(&testFileInfo{path, 100, time.Now(), nil}),
			},
		}
		src := identifier.GetSource(event)

		p.handleGrowingFingerprintLookup(logp.L(), event, src, store)

		assert.NotContains(t, p.shortFingerprintIdx.entries, oldKey, "old entry should be removed")
		require.Contains(t, p.shortFingerprintIdx.entries, newSrcID, "new entry should be added")
		assert.Equal(t, newFP, p.shortFingerprintIdx.entries[newSrcID].Fingerprint)
		assert.Equal(t, path, p.shortFingerprintIdx.entries[newSrcID].Source)
	})

	t.Run("short to max: old removed, new NOT added", func(t *testing.T) {
		const (
			oldFP = "aabb"
			path  = "/a.log"
		)
		newFP := "aabb" + strings.Repeat("0", maxEncLen-4) // starts with oldFP, len = maxEncLen
		oldKey := "filestream::input::growing_fingerprint::" + oldFP
		newSrcID := "filestream::input::growing_fingerprint::" + newFP

		store := newMockMetadataUpdater()
		store.table[oldKey] = fileMeta{Source: path, IdentifierName: growingFingerprintName}

		p := &fileProspector{
			logger:                   logp.L(),
			identifier:               identifier,
			maxEncodedFingerprintLen: maxEncLen,
			shortFingerprintIdx: &shortFingerprintIndex{maxLen: maxEncLen, entries: map[string]shortFingerprintEntry{
				oldKey: {Fingerprint: oldFP, Source: path},
			}},
		}

		event := loginp.FSEvent{
			NewPath: path,
			SrcID:   newSrcID,
			Descriptor: loginp.FileDescriptor{
				Fingerprint: newFP,
				Info:        file.ExtendFileInfo(&testFileInfo{path, 100, time.Now(), nil}),
			},
		}
		src := identifier.GetSource(event)

		p.handleGrowingFingerprintLookup(logp.L(), event, src, store)

		assert.NotContains(t, p.shortFingerprintIdx.entries, oldKey, "old entry should be removed")
		assert.NotContains(t, p.shortFingerprintIdx.entries, newSrcID, "max-length entry should NOT be added")
		assert.Len(t, p.shortFingerprintIdx.entries, 0)
	})
}

func TestShortFingerprintEntries_FullLifecycle(t *testing.T) {
	// maxEncLen=20: fingerprints with len >= 20 are "at max"
	const maxEncLen = 20

	identifier, err := newGrowingFingerprintIdentifier(nil, nil)
	require.NoError(t, err)

	makeKey := func(fp string) string {
		return "filestream::input::growing_fingerprint::" + fp
	}

	p := &fileProspector{
		logger:                   logp.L(),
		identifier:               identifier,
		maxEncodedFingerprintLen: maxEncLen,
		shortFingerprintIdx:      newShortFingerprintIndex(maxEncLen),
	}
	store := newMockMetadataUpdater()
	hg := newTestHarvesterGroup()

	// Fingerprint progressions (each step is a prefix of the next):
	//   file A: "aa" -> "aabb" -> "aabb" + padding (20 chars)
	//   file B: "bb" -> "bb" + padding (20 chars)
	//   file C: "cc" (deleted)
	fp := map[string][]string{
		"/a.log": {"aa", "aabb", "aabb" + strings.Repeat("0", maxEncLen-4)},
		"/b.log": {"bb", "bb" + strings.Repeat("0", maxEncLen-2)},
		"/c.log": {"cc"},
	}

	// --- Cycle 1: Create 3 files with short fingerprints ---
	for _, path := range []string{"/a.log", "/b.log", "/c.log"} {
		initFP := fp[path][0]
		event := loginp.FSEvent{
			Op:      loginp.OpCreate,
			NewPath: path,
			SrcID:   makeKey(initFP),
			Descriptor: loginp.FileDescriptor{
				Fingerprint: initFP,
				Info:        file.ExtendFileInfo(&testFileInfo{path, 100, time.Now(), nil}),
			},
		}
		src := identifier.GetSource(event)
		// Also seed the store with full-format keys so migration can find them
		store.table[makeKey(initFP)] = fileMeta{Source: path, IdentifierName: growingFingerprintName}
		p.onFSEvent(logp.L(), input.Context{}, event, src, store, hg, time.Time{})
	}
	assert.Equal(t, map[string]shortFingerprintEntry{
		makeKey("aa"): {Fingerprint: "aa", Source: "/a.log"},
		makeKey("bb"): {Fingerprint: "bb", Source: "/b.log"},
		makeKey("cc"): {Fingerprint: "cc", Source: "/c.log"},
	}, p.shortFingerprintIdx.entries, "cycle 1: all 3 short entries present")

	// --- Cycle 2: file A grows (still short: "aa" -> "aabb") ---
	event := loginp.FSEvent{
		Op:      loginp.OpWrite,
		NewPath: "/a.log",
		SrcID:   makeKey(fp["/a.log"][1]),
		Descriptor: loginp.FileDescriptor{
			Fingerprint: fp["/a.log"][1],
			Info:        file.ExtendFileInfo(&testFileInfo{"/a.log", 200, time.Now(), nil}),
		},
	}
	src := identifier.GetSource(event)
	p.onFSEvent(logp.L(), input.Context{}, event, src, store, hg, time.Time{})
	assert.Equal(t, map[string]shortFingerprintEntry{
		makeKey("aabb"): {Fingerprint: "aabb", Source: "/a.log"},
		makeKey("bb"):   {Fingerprint: "bb", Source: "/b.log"},
		makeKey("cc"):   {Fingerprint: "cc", Source: "/c.log"},
	}, p.shortFingerprintIdx.entries, "cycle 2: file A migrated aa->aabb, B and C unchanged")

	// --- Cycle 3: file A grows to max ("aabb" -> "aabb0000...") ---
	event = loginp.FSEvent{
		Op:      loginp.OpWrite,
		NewPath: "/a.log",
		SrcID:   makeKey(fp["/a.log"][2]),
		Descriptor: loginp.FileDescriptor{
			Fingerprint: fp["/a.log"][2],
			Info:        file.ExtendFileInfo(&testFileInfo{"/a.log", 500, time.Now(), nil}),
		},
	}
	src = identifier.GetSource(event)
	p.onFSEvent(logp.L(), input.Context{}, event, src, store, hg, time.Time{})
	assert.Equal(t, map[string]shortFingerprintEntry{
		makeKey("bb"): {Fingerprint: "bb", Source: "/b.log"},
		makeKey("cc"): {Fingerprint: "cc", Source: "/c.log"},
	}, p.shortFingerprintIdx.entries, "cycle 3: file A at max (removed), B and C remain")

	// --- Cycle 4: file B grows to max ("bb" -> "bb0000...") ---
	event = loginp.FSEvent{
		Op:      loginp.OpWrite,
		NewPath: "/b.log",
		SrcID:   makeKey(fp["/b.log"][1]),
		Descriptor: loginp.FileDescriptor{
			Fingerprint: fp["/b.log"][1],
			Info:        file.ExtendFileInfo(&testFileInfo{"/b.log", 500, time.Now(), nil}),
		},
	}
	src = identifier.GetSource(event)
	p.onFSEvent(logp.L(), input.Context{}, event, src, store, hg, time.Time{})
	assert.Equal(t, map[string]shortFingerprintEntry{
		makeKey("cc"): {Fingerprint: "cc", Source: "/c.log"},
	}, p.shortFingerprintIdx.entries, "cycle 4: file B at max (removed), only C remains")

	// --- Cycle 5: file C deleted ---
	event = loginp.FSEvent{
		Op:      loginp.OpDelete,
		OldPath: "/c.log",
		SrcID:   makeKey(fp["/c.log"][0]),
		Descriptor: loginp.FileDescriptor{
			Fingerprint: fp["/c.log"][0],
			Info:        file.ExtendFileInfo(&testFileInfo{"/c.log", 100, time.Now(), nil}),
		},
	}
	src = identifier.GetSource(event)
	p.onFSEvent(logp.L(), input.Context{}, event, src, store, hg, time.Time{})
	assert.Empty(t, p.shortFingerprintIdx.entries, "cycle 5: file C deleted, set is empty")
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
