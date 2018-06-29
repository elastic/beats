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

package file

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

const logMessage = "Test file rotator.\n"

func TestFileRotator(t *testing.T) {
	dir, err := ioutil.TempDir("", "file_rotator")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	filename := filepath.Join(dir, "sample.log")
	r, err := NewFileRotator(filename, MaxBackups(2))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	WriteMsg(t, r)
	AssertDirContents(t, dir, "sample.log")

	Rotate(t, r)
	AssertDirContents(t, dir, "sample.log.1")

	WriteMsg(t, r)
	AssertDirContents(t, dir, "sample.log", "sample.log.1")

	Rotate(t, r)
	AssertDirContents(t, dir, "sample.log.1", "sample.log.2")

	WriteMsg(t, r)
	AssertDirContents(t, dir, "sample.log", "sample.log.1", "sample.log.2")

	Rotate(t, r)
	AssertDirContents(t, dir, "sample.log.1", "sample.log.2")

	Rotate(t, r)
	AssertDirContents(t, dir, "sample.log.2")
}

func TestFileRotatorConcurrently(t *testing.T) {
	dir, err := ioutil.TempDir("", "file_rotator")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	filename := filepath.Join(dir, "sample.log")
	r, err := NewFileRotator(filename, MaxBackups(2))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	var wg sync.WaitGroup
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			defer wg.Done()
			WriteMsg(t, r)
		}()
	}
	wg.Wait()
}

func AssertDirContents(t *testing.T, dir string, files ...string) {
	t.Helper()

	f, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	names, err := f.Readdirnames(-1)
	if err != nil {
		t.Fatal(err)
	}

	sort.Strings(files)
	sort.Strings(names)
	assert.EqualValues(t, files, names)
}

func WriteMsg(t *testing.T, r *Rotator) {
	t.Helper()

	n, err := r.Write([]byte(logMessage))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(logMessage), n)
}

func Rotate(t *testing.T, r *Rotator) {
	t.Helper()

	if err := r.Rotate(); err != nil {
		t.Fatal(err)
	}
}
