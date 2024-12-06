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

package file_integrity

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScanner(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	config := defaultConfig
	config.Paths = []string{
		dir,
		filepath.Join(dir, "a"),
		"/does/not/exist",
	}

	t.Run("non-recursive", func(t *testing.T) {
		reader, err := NewFileSystemScanner(config, nil)
		if err != nil {
			t.Fatal(err)
		}

		done := make(chan struct{})
		defer close(done)

		eventC, err := reader.Start(done)
		if err != nil {
			t.Fatal(err)
		}

		var events []Event
		for event := range eventC {
			events = append(events, event)
		}
		assert.Len(t, events, 7)
	})

	t.Run("recursive", func(t *testing.T) {
		c := config
		c.Recursive = true

		reader, err := NewFileSystemScanner(c, nil)
		if err != nil {
			t.Fatal(err)
		}

		done := make(chan struct{})
		defer close(done)

		eventC, err := reader.Start(done)
		if err != nil {
			t.Fatal(err)
		}

		var foundRecursivePath bool

		var events []Event
		for event := range eventC {
			events = append(events, event)
			if filepath.Base(event.Path) == "c" {
				foundRecursivePath = true
			}
		}

		assert.Len(t, events, 8)
		assert.True(t, foundRecursivePath, "expected subdir/c to be included")
	})

	t.Run("executable", func(t *testing.T) {
		c := config
		c.FileParsers = []string{"file.elf.import_hash", "file.macho.import_hash", "file.pe.import_hash"}

		target := filepath.Join(dir, "executable")
		err := copyFile(filepath.Join("testdata", "go_pe_executable"), target)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(target)

		reader, err := NewFileSystemScanner(c, nil)
		if err != nil {
			t.Fatal(err)
		}

		done := make(chan struct{})
		defer close(done)

		eventC, err := reader.Start(done)
		if err != nil {
			t.Fatal(err)
		}

		var (
			foundExecutable bool
			events          []Event
		)
		for event := range eventC {
			events = append(events, event)
			if filepath.Base(event.Path) == "executable" {
				foundExecutable = true
				h, err := event.ParserResults.GetValue("pe.import_hash")
				assert.NoError(t, err, "no value for pe.import_hash")
				assert.Len(t, h, 16, "wrong length for hash")
			}
		}

		assert.Len(t, events, 8)
		assert.True(t, foundExecutable, "expected executable to be included")
	})

	// This smoke tests the rate limit code path, but does not validate the rate.
	t.Run("with rate limit", func(t *testing.T) {
		c := config
		c.ScanRateBytesPerSec = 1024 * 5

		reader, err := NewFileSystemScanner(c, nil)
		if err != nil {
			t.Fatal(err)
		}

		done := make(chan struct{})
		defer close(done)

		eventC, err := reader.Start(done)
		if err != nil {
			t.Fatal(err)
		}

		if err != nil {
			t.Fatal(err)
		}

		var events []Event
		for event := range eventC {
			events = append(events, event)
		}

		assert.Len(t, events, 7)
	})
}

func setupTestDir(t *testing.T) string {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("file a"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "b"), []byte("file b"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(filepath.Join(dir, "b"), filepath.Join(dir, "link_to_b")); err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "subdir", "c"), []byte("file c"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(filepath.Join(dir, "subdir"), filepath.Join(dir, "link_to_subdir")); err != nil {
		t.Fatal(err)
	}

	return dir
}

func copyFile(old, new string) error {
	o, err := os.Open(old)
	if err != nil {
		return err
	}
	defer o.Close()
	n, err := os.Create(new)
	if err != nil {
		return err
	}
	defer n.Close()
	_, err = io.Copy(n, o)
	return err
}
