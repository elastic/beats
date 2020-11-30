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
		cleanRemoved        bool
		expectedCleanedKeys []string
	}{
		"prospector init with clean_removed enabled with no entries": {
			entries:             nil,
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
			cleanRemoved:        true,
			expectedCleanedKeys: []string{"key1"},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			testStore := newMockProspectorCleaner(testCase.entries)
			p := fileProspector{identifier: mustPathIdentifier(false), cleanRemoved: testCase.cleanRemoved}
			p.Init(testStore)

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

	testCases := map[string]struct {
		entries             map[string]loginp.Value
		expectedUpdatedKeys map[string]string
	}{
		"prospector init does not update keys if there are no entries": {
			entries:             nil,
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
			expectedUpdatedKeys: map[string]string{"not_path::key1": "path::" + tmpFileName},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			testStore := newMockProspectorCleaner(testCase.entries)
			p := fileProspector{identifier: mustPathIdentifier(false)}
			p.Init(testStore)

			assert.EqualValues(t, testCase.expectedUpdatedKeys, testStore.updatedKeys)
		})
	}

}

func TestProspectorNewAndUpdatedFiles(t *testing.T) {
	minuteAgo := time.Now().Add(-1 * time.Minute)

	testCases := map[string]struct {
		events          []loginp.FSEvent
		ignoreOlder     time.Duration
		expectedSources []string
	}{
		"two new files": {
			events: []loginp.FSEvent{
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/file"},
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/other/file"},
			},
			expectedSources: []string{"path::/path/to/file", "path::/path/to/other/file"},
		},
		"one updated file": {
			events: []loginp.FSEvent{
				loginp.FSEvent{Op: loginp.OpWrite, NewPath: "/path/to/file"},
			},
			expectedSources: []string{"path::/path/to/file"},
		},
		"old files with ignore older configured": {
			events: []loginp.FSEvent{
				loginp.FSEvent{
					Op:      loginp.OpCreate,
					NewPath: "/path/to/file",
					Info:    testFileInfo{"/path/to/file", 5, minuteAgo},
				},
				loginp.FSEvent{
					Op:      loginp.OpWrite,
					NewPath: "/path/to/other/file",
					Info:    testFileInfo{"/path/to/other/file", 5, minuteAgo},
				},
			},
			ignoreOlder:     10 * time.Second,
			expectedSources: []string{},
		},
		"newer files with ignore older": {
			events: []loginp.FSEvent{
				loginp.FSEvent{
					Op:      loginp.OpCreate,
					NewPath: "/path/to/file",
					Info:    testFileInfo{"/path/to/file", 5, minuteAgo},
				},
				loginp.FSEvent{
					Op:      loginp.OpWrite,
					NewPath: "/path/to/other/file",
					Info:    testFileInfo{"/path/to/other/file", 5, minuteAgo},
				},
			},
			ignoreOlder:     5 * time.Minute,
			expectedSources: []string{"path::/path/to/file", "path::/path/to/other/file"},
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
			hg := getTestHarvesterGroup()

			p.Run(ctx, newMockMetadataUpdater(), hg)

			assert.ElementsMatch(t, hg.encounteredNames, test.expectedSources)
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
				loginp.FSEvent{Op: loginp.OpDelete, OldPath: "/path/to/file"},
			},
			cleanRemoved: false,
		},
		"one deleted file with clean removed": {
			events: []loginp.FSEvent{
				loginp.FSEvent{Op: loginp.OpDelete, OldPath: "/path/to/file"},
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

			p.Run(ctx, testStore, getTestHarvesterGroup())

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
		events                   []loginp.FSEvent
		trackRename              bool
		closeRenamed             bool
		expectedEncounteredNames []string
		expectedStoppedNames     []string
	}{
		"one renamed file without rename tracker": {
			events: []loginp.FSEvent{
				loginp.FSEvent{
					Op:      loginp.OpRename,
					OldPath: "/old/path/to/file",
					NewPath: "/new/path/to/file",
				},
			},
			expectedEncounteredNames: []string{"path::/new/path/to/file"},
			expectedStoppedNames:     []string{"path::/old/path/to/file"},
		},
		"one renamed file with rename tracker": {
			events: []loginp.FSEvent{
				loginp.FSEvent{
					Op:      loginp.OpRename,
					OldPath: "/old/path/to/file",
					NewPath: "/new/path/to/file",
				},
			},
			trackRename: true,
		},
		"one renamed file with rename tracker with close renamed": {
			events: []loginp.FSEvent{
				loginp.FSEvent{
					Op:      loginp.OpRename,
					OldPath: "/old/path/to/file",
					NewPath: "/new/path/to/file",
				},
			},
			trackRename:          true,
			closeRenamed:         true,
			expectedStoppedNames: []string{"path::/old/path/to/file"},
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

			hg := getTestHarvesterGroup()
			p.Run(ctx, testStore, hg)

			has := testStore.has("path::/old/path/to/file")
			if test.trackRename {
				assert.True(t, has)
			} else {
				assert.False(t, has)
			}

			assert.ElementsMatch(t, test.expectedEncounteredNames, hg.encounteredNames)
			assert.ElementsMatch(t, test.expectedStoppedNames, hg.stoppedNames)

		})
	}
}

type testHarvesterGroup struct {
	encounteredNames []string
	stoppedNames     []string
}

func getTestHarvesterGroup() *testHarvesterGroup {
	return &testHarvesterGroup{make([]string, 0), make([]string, 0)}
}

func (t *testHarvesterGroup) Start(_ input.Context, s loginp.Source) {
	t.encounteredNames = append(t.encounteredNames, s.Name())
}

func (t *testHarvesterGroup) Stop(_ loginp.Source) {
	return
}

func (t *testHarvesterGroup) StopGroup() error {
	return nil
}

type mockFileWatcher struct {
	nextIdx int
	events  []loginp.FSEvent
}

func (m *mockFileWatcher) Event() loginp.FSEvent {
	if len(m.events) == m.nextIdx {
		return loginp.FSEvent{}
	}
	evt := m.events[m.nextIdx]
	m.nextIdx++
	return evt
}

func (m *mockFileWatcher) Run(_ unison.Canceler) { return }

func (m *mockFileWatcher) GetFiles() map[string]os.FileInfo { return nil }

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
	v, ok := mu.table[s.Name()]
	if !ok {
		return fmt.Errorf("no such id")
	}
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
	newEntries  map[string]fileMeta
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
