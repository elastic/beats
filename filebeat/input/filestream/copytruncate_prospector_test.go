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
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
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
		"plain file rotates to GZIP": {
			events: []loginp.FSEvent{
				{Op: loginp.OpCreate, NewPath: "/path/to/log.txt",
					Descriptor: loginp.FileDescriptor{GZIP: false}},
				{Op: loginp.OpCreate, NewPath: "/path/to/log.txt.1.gz",
					Descriptor: loginp.FileDescriptor{GZIP: true}},
				{Op: loginp.OpTruncate, NewPath: "/path/to/log.txt",
					Descriptor: loginp.FileDescriptor{GZIP: false}},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/log.txt"),
				harvesterContinue("path::/path/to/log.txt -> path::/path/to/log.txt.1.gz"),
				harvesterRestart("path::/path/to/log.txt"),
				harvesterGroupStop{},
			},
			expectedRotatedFiles: map[string][]string{
				"/path/to/log.txt": {
					"/path/to/log.txt.1.gz",
				},
			},
		},
		"write to GZIP file": {
			events: []loginp.FSEvent{
				{Op: loginp.OpCreate, NewPath: "/path/to/archive.log.gz",
					Descriptor: loginp.FileDescriptor{GZIP: true}},
				{Op: loginp.OpWrite, NewPath: "/path/to/archive.log.gz",
					Descriptor: loginp.FileDescriptor{GZIP: true}},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/archive.log.gz"),
				harvesterStart("path::/path/to/archive.log.gz"),
				harvesterGroupStop{},
			},
			expectedRotatedFiles: map[string][]string{},
		},
		"truncate GZIP file": {
			events: []loginp.FSEvent{
				{Op: loginp.OpCreate, NewPath: "/path/to/archive.log.gz",
					Descriptor: loginp.FileDescriptor{GZIP: true}},
				{Op: loginp.OpTruncate, NewPath: "/path/to/archive.log.gz",
					Descriptor: loginp.FileDescriptor{GZIP: true}},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/archive.log.gz"),
				harvesterStop("path::/path/to/archive.log.gz"),
				harvesterGroupStop{},
			},
			expectedRotatedFiles: map[string][]string{},
		},
		"one new plain file, then rotated twice with GZIP in order": {
			events: []loginp.FSEvent{
				{Op: loginp.OpCreate, NewPath: "/path/to/file",
					Descriptor: loginp.FileDescriptor{GZIP: false}},
				{Op: loginp.OpCreate, NewPath: "/path/to/file.1",
					Descriptor: loginp.FileDescriptor{GZIP: true}},
				{Op: loginp.OpTruncate, NewPath: "/path/to/file",
					Descriptor: loginp.FileDescriptor{GZIP: false}},
				{Op: loginp.OpRename, NewPath: "/path/to/file.2", OldPath: "/path/to/file.1",
					Descriptor: loginp.FileDescriptor{GZIP: true}},
				{Op: loginp.OpCreate, NewPath: "/path/to/file.1",
					Descriptor: loginp.FileDescriptor{GZIP: true}},
				{Op: loginp.OpTruncate, NewPath: "/path/to/file",
					Descriptor: loginp.FileDescriptor{GZIP: false}},
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
		"one new plain file, then rotated twice with GZIP renaming": {
			events: []loginp.FSEvent{
				{Op: loginp.OpCreate, NewPath: "/path/to/file.2.gz",
					Descriptor: loginp.FileDescriptor{GZIP: true}},
				{Op: loginp.OpCreate, NewPath: "/path/to/file",
					Descriptor: loginp.FileDescriptor{GZIP: false}},
				{Op: loginp.OpCreate, NewPath: "/path/to/file.1.gz",
					Descriptor: loginp.FileDescriptor{GZIP: true}},

				{Op: loginp.OpRename,
					NewPath:    "/path/to/file.3.gz",
					OldPath:    "/path/to/file.2.gz",
					Descriptor: loginp.FileDescriptor{GZIP: true}},
				{Op: loginp.OpRename,
					NewPath:    "/path/to/file.2.gz",
					OldPath:    "/path/to/file.1.gz",
					Descriptor: loginp.FileDescriptor{GZIP: true}},

				{Op: loginp.OpCreate, NewPath: "/path/to/file.1.gz",
					Descriptor: loginp.FileDescriptor{GZIP: true}},
				{Op: loginp.OpTruncate, NewPath: "/path/to/file",
					Descriptor: loginp.FileDescriptor{GZIP: false}},
			},
			expectedEvents: []harvesterEvent{
				harvesterStart("path::/path/to/file.2.gz"),
				harvesterStart("path::/path/to/file"),
				harvesterContinue("path::/path/to/file -> path::/path/to/file.1.gz"),
				harvesterStop("path::/path/to/file.2.gz"),
				harvesterStart("path::/path/to/file.3.gz"),
				harvesterStop("path::/path/to/file.1.gz"),
				harvesterStart("path::/path/to/file.2.gz"),
				harvesterContinue("path::/path/to/file -> path::/path/to/file.1.gz"),
				harvesterRestart("path::/path/to/file"),
				harvesterGroupStop{},
			},
			expectedRotatedFiles: map[string][]string{
				"/path/to/file": {
					"/path/to/file.1.gz",
					"/path/to/file.2.gz",
					"/path/to/file.3.gz",
				},
			},
		},
	}

	for name, test := range testCases {
		test := test

		t.Run(name, func(t *testing.T) {
			s, err := newNumericSorter(`\.\d+(\.gz)?$`)
			require.NoError(t, err, "failed to create numeric sorter")

			p := copyTruncateFileProspector{
				fileProspector: fileProspector{
					filewatcher: newMockFileWatcher(test.events, len(test.events)),
					identifier:  mustPathIdentifier(false),
				},
				rotatedSuffix: regexp.MustCompile(`\.\d+(\.gz)?$`),
				rotatedFiles: &rotatedFilestreams{
					table:  make(map[string]*rotatedFilestream),
					sorter: s},
			}
			ctx := input.Context{Logger: logptest.NewTestingLogger(t, ""), Cancelation: context.Background()}
			hg := newTestHarvesterGroup()

			p.Run(ctx, newMockMetadataUpdater(), hg)

			assert.Equal(t, len(test.expectedEvents), len(hg.events))
			for i := 0; i < len(test.expectedEvents); i++ {
				assert.Equal(t, test.expectedEvents[i], hg.events[i])
			}

			for originalFile, rotatedFiles := range test.expectedRotatedFiles {
				rFile, ok := p.rotatedFiles.table[originalFile]
				if !ok {
					t.Fatalf("cannot find %s in original files\n", originalFile)
				}
				assert.Equal(t, len(rotatedFiles), len(rFile.rotated))
				for i, rotatedFile := range rotatedFiles {
					assert.Equal(t, rotatedFile, rFile.rotated[i].path,
						"%s is not a rotated file, instead %s is",
						rFile.rotated[i].path, rotatedFile)
				}
			}
		})
	}
}

func TestNumericSorter(t *testing.T) {
	type sortCase struct {
		fileinfos     []rotatedFileInfo
		expectedOrder []string
	}
	defaultSortCases := map[string]sortCase{
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

	tcs := []struct {
		name          string
		suffixRegex   string
		extraSortCase map[string]sortCase
	}{
		{name: "default"},
		{name: "custom suffix (.gz)",
			suffixRegex: `\.\d+(\.gz)?$`,
			extraSortCase: map[string]sortCase{
				"unordered fileinfos with custom suffix (.gz)": {
					fileinfos: []rotatedFileInfo{
						{path: "/path/to/apache.log.3.gz"},
						{path: "/path/to/apache.log.1.gz"},
						{path: "/path/to/apache.log.2.gz"},
					},
					expectedOrder: []string{
						"/path/to/apache.log.1.gz",
						"/path/to/apache.log.2.gz",
						"/path/to/apache.log.3.gz",
					},
				},
			},
		}}

	for _, test := range tcs {
		sorter, _ := newNumericSorter(test.suffixRegex)
		sortCases := map[string]sortCase{}
		for k, v := range defaultSortCases {
			sortCases[k] = v
		}
		for k, v := range test.extraSortCase {
			sortCases[k] = v
		}

		t.Run(test.name, func(t *testing.T) {
			for name, sortcase := range sortCases {
				t.Run(name, func(t *testing.T) {
					sorter.sort(sortcase.fileinfos)
					var fisPath []string
					for _, fi := range sortcase.fileinfos {
						fisPath = append(fisPath, fi.path)
					}
					assert.Equal(t, sortcase.expectedOrder, fisPath)
				})
			}
		})
	}
}

func TestDateSorter(t *testing.T) {
	testCases := map[string]struct {
		fileinfos     []rotatedFileInfo
		expectedOrder []string

		dateRegex   string
		dateFormat  string
		suffixRegex string
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
		"unordered fileinfos with custom suffix (.gz), regex without .gz": {
			fileinfos: []rotatedFileInfo{
				{path: "/path/to/apache.log-20140507.gz"},
				{path: "/path/to/apache.log-20140508.gz"},
				{path: "/path/to/apache.log-20140506.gz"},
			},
			expectedOrder: []string{
				"/path/to/apache.log-20140508.gz",
				"/path/to/apache.log-20140507.gz",
				"/path/to/apache.log-20140506.gz",
			},
		},
		"unordered fileinfos with custom suffix (.gz), regex with .gz": {
			fileinfos: []rotatedFileInfo{
				{path: "/path/to/apache.log-20140507.gz"},
				{path: "/path/to/apache.log-20140508.gz"},
				{path: "/path/to/apache.log-20140506.gz"},
			},
			expectedOrder: []string{
				"/path/to/apache.log-20140508.gz",
				"/path/to/apache.log-20140507.gz",
				"/path/to/apache.log-20140506.gz",
			},
			dateFormat: `-20060102`,
			// date is 2nd match of the regex, in other words, the 1st capturing
			// group
			dateRegex:   `(-\d{8})(\.gz)?`,
			suffixRegex: `-\d{8}(\.gz)?$`,
		},
	}

	for name, test := range testCases {
		test := test
		t.Run(name, func(t *testing.T) {
			format := test.dateFormat
			dateRexex := test.dateRegex
			suffixRexex := test.suffixRegex

			if format == "" {
				format = "-20060102"
			}
			if dateRexex == "" {
				dateRexex = `-\d{8}`
			}
			if suffixRexex == "" {
				suffixRexex = `-\d{8}?(\.gz)?$`
			}
			sorter, err := newDateSorter(
				format,
				dateRexex,
				suffixRexex)
			require.NoError(t, err, "failed to create date sorter")

			sorter.sort(test.fileinfos)
			var fisPath []string
			for _, fi := range test.fileinfos {
				fisPath = append(fisPath, fi.path)
			}
			assert.Equal(t, test.expectedOrder, fisPath)
		})
	}
}
