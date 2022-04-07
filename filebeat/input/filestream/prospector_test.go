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

//nolint: errcheck // It's a test file
package filestream

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/unison"
)

func TestProspector_InitCleanIfRemoved(t *testing.T) {
	testCases := map[string]struct {
		entries             map[string]loginp.Value
		filesOnDisk         map[string]os.FileInfo
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
					fileMeta{
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
					fileMeta{
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
			testStore := newMockProspectorCleaner(testCase.entries)
			p := fileProspector{
				identifier:   mustPathIdentifier(false),
				cleanRemoved: testCase.cleanRemoved,
				filewatcher:  &mockFileWatcher{filesOnDisk: testCase.filesOnDisk},
			}
			p.Init(testStore, newMockProspectorCleaner(nil), func(loginp.Source) string { return "" })

			assert.ElementsMatch(t, testCase.expectedCleanedKeys, testStore.cleanedKeys)
		})
	}
}

func TestProspector_InitUpdateIdentifiers(t *testing.T) {
	f, err := ioutil.TempFile("", "existing_file")
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
		filesOnDisk         map[string]os.FileInfo
		expectedUpdatedKeys map[string]string
	}{
		"prospector init does not update keys if there are no entries": {
			entries:             nil,
			filesOnDisk:         nil,
			expectedUpdatedKeys: map[string]string{},
		},
		"prospector init does not update keys of not existing files": {
			entries: map[string]loginp.Value{
				"not_path::key1": &mockUnpackValue{
					fileMeta{
						Source:         "/no/such/path",
						IdentifierName: "not_path",
					},
				},
			},
			filesOnDisk:         nil,
			expectedUpdatedKeys: map[string]string{},
		},
		"prospector init updates keys of existing files": {
			entries: map[string]loginp.Value{
				"not_path::key1": &mockUnpackValue{
					fileMeta{
						Source:         tmpFileName,
						IdentifierName: "not_path",
					},
				},
			},
			filesOnDisk: map[string]os.FileInfo{
				tmpFileName: fi,
			},
			expectedUpdatedKeys: map[string]string{"not_path::key1": "path::" + tmpFileName},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase

		t.Run(name, func(t *testing.T) {
			testStore := newMockProspectorCleaner(testCase.entries)
			p := fileProspector{
				identifier:  mustPathIdentifier(false),
				filewatcher: &mockFileWatcher{filesOnDisk: testCase.filesOnDisk},
			}
			p.Init(testStore, newMockProspectorCleaner(nil), func(loginp.Source) string { return "" })

			assert.EqualValues(t, testCase.expectedUpdatedKeys, testStore.updatedKeys)
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
				{Op: loginp.OpCreate, NewPath: "/path/to/file", Info: testFileInfo{}},
				{Op: loginp.OpCreate, NewPath: "/path/to/other/file", Info: testFileInfo{}},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/file"),
				harvesterStart("path::/path/to/other/file"),
				harvesterGroupStop{},
			},
		},
		"one updated file": {
			events: []loginp.FSEvent{
				{Op: loginp.OpWrite, NewPath: "/path/to/file", Info: testFileInfo{}},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/file"),
				harvesterGroupStop{},
			},
		},
		"one updated then truncated file": {
			events: []loginp.FSEvent{
				{Op: loginp.OpWrite, NewPath: "/path/to/file", Info: testFileInfo{}},
				{Op: loginp.OpTruncate, NewPath: "/path/to/file", Info: testFileInfo{}},
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
					Op:      loginp.OpCreate,
					NewPath: "/path/to/file",
					Info:    testFileInfo{"/path/to/file", 5, minuteAgo, nil},
				},
				{
					Op:      loginp.OpWrite,
					NewPath: "/path/to/other/file",
					Info:    testFileInfo{"/path/to/other/file", 5, minuteAgo, nil},
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
					Op:      loginp.OpCreate,
					NewPath: "/path/to/file",
					Info:    testFileInfo{"/path/to/file", 5, minuteAgo, nil},
				},
				{
					Op:      loginp.OpWrite,
					NewPath: "/path/to/other/file",
					Info:    testFileInfo{"/path/to/other/file", 5, minuteAgo, nil},
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
				filewatcher: &mockFileWatcher{events: test.events},
				identifier:  mustPathIdentifier(false),
				ignoreOlder: test.ignoreOlder,
			}
			ctx := input.Context{Logger: logp.L(), Cancelation: context.Background()}
			hg := newTestHarvesterGroup()

			p.Run(ctx, newMockMetadataUpdater(), hg)

			assert.ElementsMatch(t, test.expectedEvents, hg.events)
		})
	}
}

func TestProspectorDeletedFile(t *testing.T) {
	testCases := map[string]struct {
		events       []loginp.FSEvent
		cleanRemoved bool
	}{
		"one deleted file without clean removed": {
			events: []loginp.FSEvent{
				{Op: loginp.OpDelete, OldPath: "/path/to/file", Info: testFileInfo{}},
			},
			cleanRemoved: false,
		},
		"one deleted file with clean removed": {
			events: []loginp.FSEvent{
				{Op: loginp.OpDelete, OldPath: "/path/to/file", Info: testFileInfo{}},
			},
			cleanRemoved: true,
		},
	}

	for name, test := range testCases {
		test := test

		t.Run(name, func(t *testing.T) {
			p := fileProspector{
				filewatcher:  &mockFileWatcher{events: test.events},
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
					Op:      loginp.OpRename,
					OldPath: "/old/path/to/file",
					NewPath: "/new/path/to/file",
					Info:    testFileInfo{},
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
					Op:      loginp.OpRename,
					OldPath: "/old/path/to/file",
					NewPath: "/new/path/to/file",
					Info:    testFileInfo{},
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
					Op:      loginp.OpRename,
					OldPath: "/old/path/to/file",
					NewPath: "/new/path/to/file",
					Info:    testFileInfo{},
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
				filewatcher:       &mockFileWatcher{events: test.events},
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

func (t *testHarvesterGroup) StopGroup() error {
	t.events = append(t.events, harvesterGroupStop{})
	return nil
}

type mockFileWatcher struct {
	nextIdx     int
	events      []loginp.FSEvent
	filesOnDisk map[string]os.FileInfo
}

func (m *mockFileWatcher) Event() loginp.FSEvent {
	if len(m.events) == m.nextIdx {
		return loginp.FSEvent{}
	}
	evt := m.events[m.nextIdx]
	m.nextIdx++
	return evt
}

func (m *mockFileWatcher) Run(_ unison.Canceler) {}

func (m *mockFileWatcher) GetFiles() map[string]os.FileInfo { return m.filesOnDisk }

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

func (mu *mockMetadataUpdater) FindCursorMeta(s loginp.Source, v interface{}) error {
	_, ok := mu.table[s.Name()]
	if !ok {
		return fmt.Errorf("no such id")
	}
	return nil
}

func (mu *mockMetadataUpdater) ResetCursor(s loginp.Source, cur interface{}) error {
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
}

func (u *mockUnpackValue) UnpackCursorMeta(to interface{}) error {
	return typeconv.Convert(to, u.fileMeta)
}

type mockProspectorCleaner struct {
	available   map[string]loginp.Value
	cleanedKeys []string
	updatedKeys map[string]string
}

func newMockProspectorCleaner(available map[string]loginp.Value) *mockProspectorCleaner {
	return &mockProspectorCleaner{
		available:   available,
		cleanedKeys: make([]string, 0),
		updatedKeys: make(map[string]string, 0),
	}
}

func (c *mockProspectorCleaner) CleanIf(pred func(v loginp.Value) bool) {
	for key, meta := range c.available {
		if pred(meta) {
			c.cleanedKeys = append(c.cleanedKeys, key)
		}
	}
}

func (c *mockProspectorCleaner) UpdateIdentifiers(updater func(v loginp.Value) (string, interface{})) {
	for key, meta := range c.available {
		k, _ := updater(meta)
		if k != "" {
			c.updatedKeys[key] = k
		}
	}
}

// FixUpIdentifiers does nothing
func (c *mockProspectorCleaner) FixUpIdentifiers(func(loginp.Value) (string, interface{})) {}

type renamedPathIdentifier struct {
	fileIdentifier
}

func (p *renamedPathIdentifier) Supports(_ identifierFeature) bool { return true }

func mustPathIdentifier(renamed bool) fileIdentifier {
	pathIdentifier, err := newPathIdentifier(nil)
	if err != nil {
		panic(err)
	}
	if renamed {
		return &renamedPathIdentifier{pathIdentifier}
	}
	return pathIdentifier
}
