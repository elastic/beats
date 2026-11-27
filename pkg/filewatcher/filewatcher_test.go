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
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFileWatcher(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		"file-1",
		"file-2",
		"file-3",
	}
	filenames := []string{}

	// Create the files and write something on them
	for _, f := range files {
		filename := filepath.Join(dir, f)
		if err := os.WriteFile(filename, []byte("test\n"), 0644); err != nil {
			t.Fatalf("could not create '%s' for testing, err: %s", filename, err)
		}
		filenames = append(filenames, filename)
	}

	watcher := New(filenames...)

	// Modification timestamps usually have second precision,
	// we wait to make sure we're not in the second the files were created
	time.Sleep(2 * time.Second)

	files, changed, err := watcher.Scan()
	assert.Len(t, files, 3, "number of watched files")
	assert.NoError(t, err)
	assert.True(t, changed, "first scan should always return true for 'changed'")

	files, changed, err = watcher.Scan()
	assert.Len(t, files, 3, "number of watched files")
	assert.NoError(t, err)
	assert.False(t, changed, "'changed' must be false, no files should have changed")

	// Modify one file
	err = os.WriteFile(filenames[2], []byte("data\n"), 0644)
	assert.NoError(t, err)

	files, changed, err = watcher.Scan()
	assert.Len(t, files, 3, "number of files watched")
	assert.NoError(t, err)
	assert.True(t, changed, "'changed' must be true, one file has changed")

	// Remove a file
	err = os.Remove(filenames[2])
	assert.NoError(t, err)

	files, changed, err = watcher.Scan()
	assert.Len(t, files, 2, "number of files watched")
	assert.NoError(t, err)
	assert.True(t, changed, "'changed' must be true, one file has been removed")
}

func TestHash(t *testing.T) {
	files := []string{"file-1", "file-2", "file-3"}
	i, err := hash(files)
	assert.NoError(t, err)
	// ensure custom hash function returns the same result as deprecated hashstructure lib
	assert.Equal(t, uint64(11400963159482616226), i)
}
