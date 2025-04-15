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

//go:build !requirefips

package file_integrity

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Scanner_Executable(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	c := defaultConfig
	c.Paths = []string{
		dir,
		filepath.Join(dir, "a"),
		"/does/not/exist",
	}
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
}
