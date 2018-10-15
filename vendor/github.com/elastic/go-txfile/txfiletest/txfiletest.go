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

// Package txfiletest provides utilities for testing on top of txfile.
package txfiletest

import (
	"os"

	"github.com/elastic/go-txfile"
	"github.com/elastic/go-txfile/internal/cleanup"
	"github.com/elastic/go-txfile/internal/vfs/osfs/osfstest"
)

// TestFile wraps a txfile.File structure for testing.
type TestFile struct {
	*txfile.File
	t    testT
	Path string
	opts txfile.Options
}

type testT interface {
	Error(...interface{})
	Fatal(...interface{})
}

// SetupTestFile creates a new testfile in a temporary directory.
// The teardown function will remove the directory and the temporary file.
func SetupTestFile(t testT, opts txfile.Options) (tf *TestFile, teardown func()) {
	if opts.PageSize == 0 {
		opts.PageSize = 4096
	}

	ok := false
	path, cleanPath := SetupPath(t, "")
	defer cleanup.IfNot(&ok, cleanPath)

	tf = &TestFile{Path: path, t: t, opts: opts}
	tf.Open()

	ok = true
	return tf, func() {
		tf.Close()
		cleanPath()
	}
}

// Reopen tries to close and open the file again.
func (f *TestFile) Reopen() {
	f.Close()
	f.Open()
}

// Close the test file.
func (f *TestFile) Close() {
	if f.File != nil {
		if err := f.File.Close(); err != nil {
			f.t.Fatal("close failed on reopen")
		}
		f.File = nil
	}
}

// Open opens the file if it has been closed.
// The File pointer will be changed.
func (f *TestFile) Open() {
	if f.File != nil {
		return
	}

	tmp, err := txfile.Open(f.Path, os.ModePerm, f.opts)
	if err != nil {
		f.t.Fatal("open failed:", err)
	}
	f.File = tmp
}

// SetupPath creates a temporary directory for testing.
// Use the teardown function to remove the directory again.
func SetupPath(t testT, file string) (dir string, teardown func()) {
	return osfstest.SetupPath(t, file)
}
