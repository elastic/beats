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

package filewatcher

import (
	"os"
	"time"

	"github.com/mitchellh/hashstructure"

	"github.com/elastic/elastic-agent-libs/logp"
)

type FileWatcher struct {
	files    []string
	lastScan time.Time
	lastHash uint64
}

func New(files ...string) *FileWatcher {
	return &FileWatcher{
		lastScan: time.Time{},
		lastHash: 0,
		files:    files,
	}
}

// Scan scans all file paths and checks if the number of files or the modtime of the files changed
// It returns the list of existing files, a boolean if anything in has changed and potential errors.
// To detect changes not only modtime is compared but also the hash of the files list. This is required to
// also detect files which were removed.
// Normally, the modtime is presented in seconds, so the change detection is also based on seconds.
// When it's unclear whether something changed or not the method will return `true` to make sure potential changes are handled.
// It is strongly recommended to call `Scan` not more than once a second.
func (f *FileWatcher) Scan() ([]string, bool, error) {
	updatedFiles := false
	files := []string{}

	lastScan := time.Now()
	defer func() { f.lastScan = lastScan }()

	for _, path := range f.files {
		info, err := os.Stat(path)
		if err != nil {
			logp.Err("Error getting stats for file: %s", path)
			continue
		}

		// Check if one of the files was changed recently
		// File modification time usually is in seconds. We subtract a
		// second and truncate to account for files which  were
		// created during this second the scan is running.
		// If the last scan was at 09:02:15.00001 it will pick up
		// files which were modified at 09:02:14.
		// Otherwise this scan would not necessarily pick up files
		// form 09:02:14.
		// TODO: How could this be improved / simplified? Behaviour was sometimes flaky. Is ModTime updated with delay?
		if info.ModTime().After(f.lastScan.Add(-1 * time.Second).Truncate(time.Second)) {
			updatedFiles = true
		}

		files = append(files, path)
	}

	hash, err := hashstructure.Hash(files, nil)
	if err != nil {
		return files, true, err
	}
	defer func() { f.lastHash = hash }()

	// Check if something changed
	if !updatedFiles && hash == f.lastHash {
		return files, false, nil
	}

	return files, true, nil
}
