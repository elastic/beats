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
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestCopyTruncateProspector_Create(t *testing.T) {
	testCases := map[string]struct {
		events               []loginp.FSEvent
		expectedEvents       []harvesterEvent
		expectedRotatedFiles map[string][]string
	}{
		"one new file, then rotated": {
			events: []loginp.FSEvent{
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/file"},
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/file.1"},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/file"),
				harvesterContinue("path::/path/to/file.1"),
				harvesterGroupStop{},
			},
			expectedRotatedFiles: map[string][]string{
				"/path/to/file": []string{"/path/to/file.1"},
			},
		},
		"one new file, then rotated twice in order": {
			events: []loginp.FSEvent{
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/file"},
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/file.1"},
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/file.2"},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/file"),
				harvesterContinue("path::/path/to/file.1"),
				harvesterContinue("path::/path/to/file.2"),
				harvesterGroupStop{},
			},
			expectedRotatedFiles: map[string][]string{
				"/path/to/file": []string{"/path/to/file.1", "/path/to/file.2"},
			},
		},
		"one new file, then rotated twice unordered": {
			events: []loginp.FSEvent{
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/file"},
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/file.2"},
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/file.1"},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/file"),
				harvesterContinue("path::/path/to/file.1"),
				harvesterContinue("path::/path/to/file.2"),
				harvesterGroupStop{},
			},
			expectedRotatedFiles: map[string][]string{
				"/path/to/file": []string{"/path/to/file.1", "/path/to/file.2"},
			},
		},
		"first rotated file, when rotated file not exist": {
			events: []loginp.FSEvent{
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/file.1"},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/file.1"),
				harvesterGroupStop{},
			},
			expectedRotatedFiles: map[string][]string{},
		},
	}

	for name, test := range testCases {
		test := test

		t.Run(name, func(t *testing.T) {
			p := copyTruncateFileProspector{
				fileProspector{
					filewatcher: &mockFileWatcher{events: test.events},
					identifier:  mustPathIdentifier(false),
				},
				regexp.MustCompile("\\.\\d$"),
				make(map[string]*rotatedFileGroup),
			}
			ctx := input.Context{Logger: logp.L(), Cancelation: context.Background()}
			hg := newTestHarvesterGroup()

			p.Run(ctx, newMockMetadataUpdater(), hg)

			assert.ElementsMatch(t, test.expectedEvents, hg.events)

			for originalFile, rotatedFiles := range test.expectedRotatedFiles {
				rFiles, ok := p.rotatedFiles[originalFile]
				if !ok {
					fmt.Printf("cannot find %s in original file\n", originalFile)
					t.FailNow()
				}
				assert.Equal(t, len(rFiles.rotated), len(rotatedFiles))
				for i := 0; i < len(rotatedFiles); i++ {
					if rFiles.rotated[i].path != rotatedFiles[i] {
						fmt.Printf("cannot find %s in actual rotated files\n", rotatedFiles[i])
						t.FailNow()
					}
				}
			}
		})
	}
}

func TestCopyTruncateProspector_Delete(t *testing.T) {
	testCases := map[string]struct {
		events               []loginp.FSEvent
		rotated              map[string]*rotatedFileGroup
		expectedRotatedFiles map[string][]string
	}{
		"remove last rotated file": {
			events: []loginp.FSEvent{
				loginp.FSEvent{Op: loginp.OpDelete, OldPath: "/path/to/file.1"},
			},
			rotated: map[string]*rotatedFileGroup{
				"/path/to/file": &rotatedFileGroup{rotated: []rotatedFileInfo{rotatedFileInfo{path: "/path/to/file.1"}}},
			},
			expectedRotatedFiles: map[string][]string{
				"/path/to/file": []string{},
			},
		},
	}

	for name, test := range testCases {
		test := test

		t.Run(name, func(t *testing.T) {
			p := copyTruncateFileProspector{
				fileProspector{
					filewatcher: &mockFileWatcher{events: test.events},
					identifier:  mustPathIdentifier(false),
				},
				regexp.MustCompile("\\.\\d$"),
				test.rotated,
			}
			ctx := input.Context{Logger: logp.L(), Cancelation: context.Background()}
			hg := newTestHarvesterGroup()

			p.Run(ctx, newMockMetadataUpdater(), hg)

			for originalFile, rotatedFiles := range test.expectedRotatedFiles {
				rFiles, ok := p.rotatedFiles[originalFile]
				if !ok {
					fmt.Printf("cannot find %s in original file\n", originalFile)
					t.FailNow()
				}
				assert.Equal(t, len(rFiles.rotated), len(rotatedFiles))
				for i := 0; i < len(rotatedFiles); i++ {
					if rFiles.rotated[i].path != rotatedFiles[i] {
						fmt.Printf("cannot find %s in actual rotated files\n", rotatedFiles[i])
						t.FailNow()
					}
				}
			}
		})
	}
}
