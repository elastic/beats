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
	meta, ok := mu.table[s.Name()]
	if !ok {
		return fmt.Errorf("no such id [%q]", s.Name())
	}
	return typeconv.Convert(v, meta)
}

func (mu *mockMetadataUpdater) ResetCursor(s loginp.Source, cur interface{}) error {
	mu.table[s.Name()] = cur
	return nil
}

func (mu *mockMetadataUpdater) UpdateMetadata(s loginp.Source, v interface{}) error {
	mu.table[s.Name()] = v
	return nil
}

func (mu *mockMetadataUpdater) Remove(s loginp.Source) error {
	delete(mu.table, s.Name())
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
