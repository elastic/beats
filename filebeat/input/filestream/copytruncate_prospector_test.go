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
		events              []loginp.FSEvent
		expectedEvents      []harvesterEvent
		expectedRotatedFile map[string]string
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
			expectedRotatedFile: map[string]string{
				"/path/to/file": "/path/to/file.1",
			},
		},
		"one new file, then rotated twice in order": {
			events: []loginp.FSEvent{
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/file"},
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/file.1"},
				loginp.FSEvent{Op: loginp.OpTruncate, NewPath: "/path/to/file"},
				loginp.FSEvent{Op: loginp.OpRename, NewPath: "/path/to/file.2", OldPath: "/path/to/file.1"},
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/file.1"},
				loginp.FSEvent{Op: loginp.OpTruncate, NewPath: "/path/to/file"},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/file"),
				harvesterContinue("path::/path/to/file.1"),
				harvesterRestart("path::/path/to/file"),
				harvesterContinue("path::/path/to/file.1"),
				harvesterRestart("path::/path/to/file"),
				harvesterGroupStop{},
			},
			expectedRotatedFile: map[string]string{
				"/path/to/file": "/path/to/file.1",
			},
		},
		"one new file, then rotated twice unordered": {
			events: []loginp.FSEvent{
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/file"},
				loginp.FSEvent{Op: loginp.OpRename, NewPath: "/path/to/file.2", OldPath: "/path/to/file.1"},
				loginp.FSEvent{Op: loginp.OpCreate, NewPath: "/path/to/file.1"},
				loginp.FSEvent{Op: loginp.OpTruncate, NewPath: "/path/to/file"},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/file"),
				harvesterContinue("path::/path/to/file.1"),
				harvesterRestart("path::/path/to/file"),
				harvesterGroupStop{},
			},
			expectedRotatedFile: map[string]string{
				"/path/to/file": "/path/to/file.1",
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
			expectedRotatedFile: map[string]string{},
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
				rotatedFilestreams{make(map[string]*rotatedFilestream)},
			}
			ctx := input.Context{Logger: logp.L(), Cancelation: context.Background()}
			hg := newTestHarvesterGroup()

			p.Run(ctx, newMockMetadataUpdater(), hg)

			assert.ElementsMatch(t, test.expectedEvents, hg.events)

			for originalFile, rotatedFile := range test.expectedRotatedFile {
				rFile, ok := p.rotatedFiles.table[originalFile]
				if !ok {
					fmt.Printf("cannot find %s in original files\n", originalFile)
					t.FailNow()
				}
				if rFile.rotated.path != rotatedFile {
					fmt.Printf("%s is not a rotated file, instead %s is\n", rFile.rotated.path, rotatedFile)
					t.FailNow()
				}
			}
		})
	}
}
