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

	"github.com/stretchr/testify/require"

	loginp "github.com/menderesk/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/menderesk/beats/v7/filebeat/input/v2"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

func TestCopyTruncateProspector_Create(t *testing.T) {
	testCases := map[string]struct {
		events               []loginp.FSEvent
		expectedEvents       []harvesterEvent
		expectedRotatedFiles map[string][]string
	}{
		"one new file, then rotated": {
			events: []loginp.FSEvent{
				{Op: loginp.OpCreate, NewPath: "/path/to/file"},
				{Op: loginp.OpCreate, NewPath: "/path/to/file.1"},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/file"),
				harvesterContinue("path::/path/to/file -> path::/path/to/file.1"),
				harvesterGroupStop{},
			},
			expectedRotatedFiles: map[string][]string{
				"/path/to/file": {
					"/path/to/file.1",
				},
			},
		},
		"one new file, then rotated twice in order": {
			events: []loginp.FSEvent{
				{Op: loginp.OpCreate, NewPath: "/path/to/file"},
				{Op: loginp.OpCreate, NewPath: "/path/to/file.1"},
				{Op: loginp.OpTruncate, NewPath: "/path/to/file"},
				{Op: loginp.OpRename, NewPath: "/path/to/file.2", OldPath: "/path/to/file.1"},
				{Op: loginp.OpCreate, NewPath: "/path/to/file.1"},
				{Op: loginp.OpTruncate, NewPath: "/path/to/file"},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/file"),
				harvesterContinue("path::/path/to/file -> path::/path/to/file.1"),
				harvesterRestart("path::/path/to/file"),
				harvesterStop("path::/path/to/file.1"),
				harvesterStart("path::/path/to/file.2"),
				harvesterContinue("path::/path/to/file -> path::/path/to/file.1"),
				harvesterRestart("path::/path/to/file"),
				harvesterGroupStop{},
			},
			expectedRotatedFiles: map[string][]string{
				"/path/to/file": {
					"/path/to/file.1",
					"/path/to/file.2",
				},
			},
		},
		"one new file, then rotated twice with renaming": {
			events: []loginp.FSEvent{
				{Op: loginp.OpCreate, NewPath: "/path/to/file.2"},
				{Op: loginp.OpCreate, NewPath: "/path/to/file"},
				{Op: loginp.OpCreate, NewPath: "/path/to/file.1"},
				{Op: loginp.OpRename, NewPath: "/path/to/file.3", OldPath: "/path/to/file.2"},
				{Op: loginp.OpRename, NewPath: "/path/to/file.2", OldPath: "/path/to/file.1"},
				{Op: loginp.OpCreate, NewPath: "/path/to/file.1"},
				{Op: loginp.OpTruncate, NewPath: "/path/to/file"},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/file.2"),
				harvesterStart("path::/path/to/file"),
				harvesterContinue("path::/path/to/file -> path::/path/to/file.1"),
				harvesterStop("path::/path/to/file.2"),
				harvesterStart("path::/path/to/file.3"),
				harvesterStop("path::/path/to/file.1"),
				harvesterStart("path::/path/to/file.2"),
				harvesterContinue("path::/path/to/file -> path::/path/to/file.1"),
				harvesterRestart("path::/path/to/file"),
				harvesterGroupStop{},
			},
			expectedRotatedFiles: map[string][]string{
				"/path/to/file": {
					"/path/to/file.1",
					"/path/to/file.2",
					"/path/to/file.3",
				},
			},
		},
		"first rotated file, when rotated file not exist": {
			events: []loginp.FSEvent{
				{Op: loginp.OpCreate, NewPath: "/path/to/file.1"},
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
				&rotatedFilestreams{make(map[string]*rotatedFilestream), newNumericSorter()},
			}
			ctx := input.Context{Logger: logp.L(), Cancelation: context.Background()}
			hg := newTestHarvesterGroup()

			p.Run(ctx, newMockMetadataUpdater(), hg)

			require.Equal(t, len(test.expectedEvents), len(hg.events))
			for i := 0; i < len(test.expectedEvents); i++ {
				require.Equal(t, test.expectedEvents[i], hg.events[i])
			}

			for originalFile, rotatedFiles := range test.expectedRotatedFiles {
				rFile, ok := p.rotatedFiles.table[originalFile]
				if !ok {
					fmt.Printf("cannot find %s in original files\n", originalFile)
					t.FailNow()
				}
				require.Equal(t, len(rotatedFiles), len(rFile.rotated))
				for i, rotatedFile := range rotatedFiles {
					if rFile.rotated[i].path != rotatedFile {
						fmt.Printf("%s is not a rotated file, instead %s is\n", rFile.rotated[i].path, rotatedFile)
						t.FailNow()
					}
				}
			}
		})
	}
}

func TestNumericSorter(t *testing.T) {
	testCases := map[string]struct {
		fileinfos     []rotatedFileInfo
		expectedOrder []string
	}{
		"one fileinfo": {
			fileinfos: []rotatedFileInfo{
				{path: "/path/to/apache.log.1"},
			},
			expectedOrder: []string{
				"/path/to/apache.log.1",
			},
		},
		"ordered fileinfos": {
			fileinfos: []rotatedFileInfo{
				{path: "/path/to/apache.log.1"},
				{path: "/path/to/apache.log.2"},
				{path: "/path/to/apache.log.3"},
			},
			expectedOrder: []string{
				"/path/to/apache.log.1",
				"/path/to/apache.log.2",
				"/path/to/apache.log.3",
			},
		},
		"unordered fileinfos": {
			fileinfos: []rotatedFileInfo{
				{path: "/path/to/apache.log.3"},
				{path: "/path/to/apache.log.1"},
				{path: "/path/to/apache.log.2"},
			},
			expectedOrder: []string{
				"/path/to/apache.log.1",
				"/path/to/apache.log.2",
				"/path/to/apache.log.3",
			},
		},
		"unordered fileinfos with numbers in filename": {
			fileinfos: []rotatedFileInfo{
				{path: "/path/to/apache42.log.3"},
				{path: "/path/to/apache43.log.1"},
				{path: "/path/to/apache44.log.2"},
			},
			expectedOrder: []string{
				"/path/to/apache43.log.1",
				"/path/to/apache44.log.2",
				"/path/to/apache42.log.3",
			},
		},
	}
	sorter := newNumericSorter()

	for name, test := range testCases {
		test := test
		t.Run(name, func(t *testing.T) {
			sorter.sort(test.fileinfos)
			for i, fi := range test.fileinfos {
				require.Equal(t, test.expectedOrder[i], fi.path)
			}
		})
	}
}

func TestDateSorter(t *testing.T) {
	testCases := map[string]struct {
		fileinfos     []rotatedFileInfo
		expectedOrder []string
	}{
		"one fileinfo": {
			fileinfos: []rotatedFileInfo{
				{path: "/path/to/apache.log-20140506"},
			},
			expectedOrder: []string{
				"/path/to/apache.log-20140506",
			},
		},
		"ordered fileinfos": {
			fileinfos: []rotatedFileInfo{
				{path: "/path/to/apache.log-20140506"},
				{path: "/path/to/apache.log-20140507"},
				{path: "/path/to/apache.log-20140508"},
			},
			expectedOrder: []string{
				"/path/to/apache.log-20140508",
				"/path/to/apache.log-20140507",
				"/path/to/apache.log-20140506",
			},
		},
		"unordered fileinfos": {
			fileinfos: []rotatedFileInfo{
				{path: "/path/to/apache.log-20140507"},
				{path: "/path/to/apache.log-20140508"},
				{path: "/path/to/apache.log-20140506"},
			},
			expectedOrder: []string{
				"/path/to/apache.log-20140508",
				"/path/to/apache.log-20140507",
				"/path/to/apache.log-20140506",
			},
		},
	}
	sorter := dateSorter{"-20060102"}

	for name, test := range testCases {
		test := test
		t.Run(name, func(t *testing.T) {
			sorter.sort(test.fileinfos)
			for i, fi := range test.fileinfos {
				require.Equal(t, test.expectedOrder[i], fi.path)
			}
		})
	}
}
