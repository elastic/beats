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
	"testing"

	"github.com/stretchr/testify/assert"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	"github.com/elastic/go-concert/unison"
)

func TestProspectorNewAndUpdatedFiles(t *testing.T) {
	testCases := map[string]struct {
		events              []loginp.FSEvent
		expectedSourceNames []string
	}{
		"two new files": {
			events: []loginp.FSEvent{
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/file"},
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/other/file"},
			},
			expectedSourceNames: []string{"filestream::path::/path/to/file", "filestream::path::/path/to/other/file"},
		},
		"one updated file": {
			events: []loginp.FSEvent{
				loginp.FSEvent{Op: loginp.OpWrite, NewPath: "/path/to/file"},
			},
			expectedSourceNames: []string{"filestream::path::/path/to/file"},
		},
	}

	for name, test := range testCases {
		test := test

		t.Run(name, func(t *testing.T) {
			pathIdentifier, err := newPathIdentifier(nil)
			if err != nil {
				t.Fatal(err)
			}

			p := fileProspector{
				filewatcher: &mockFileWatcher{events: test.events},
				identifier:  pathIdentifier,
			}

			ctx := input.Context{Logger: logp.L(), Cancelation: context.Background()}
			hg := &testHarvesterGroup{t: t, expectedNames: test.expectedSourceNames}

			p.Run(ctx, testStateStore(), hg)
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
			pathIdentifier, err := newPathIdentifier(nil)
			if err != nil {
				t.Fatal(err)
			}

			p := fileProspector{
				filewatcher:  &mockFileWatcher{events: test.events},
				identifier:   pathIdentifier,
				cleanRemoved: test.cleanRemoved,
			}

			ctx := input.Context{Logger: logp.L(), Cancelation: context.Background()}
			testStore := testStateStore()
			testStore.Set("filestream::path::/path/to/file", nil)

			p.Run(ctx, testStore, &testHarvesterGroup{})

			has, err := testStore.Has("filestream::path::/path/to/file")
			if err != nil {
				t.Fatal(err)
			}

			if test.cleanRemoved {
				assert.False(t, has)
			} else {
				assert.True(t, has)

			}
		})
	}
}

type testHarvesterGroup struct {
	t             *testing.T
	expectedNames []string
}

func (t *testHarvesterGroup) Run(_ input.Context, s loginp.Source) error {
	assert.Contains(t.t, t.expectedNames, s.Name())
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

func testStateStore() *statestore.Store {
	s, _ := statestore.NewRegistry(storetest.NewMemoryStoreBackend()).Get(pluginName)
	return s
}
