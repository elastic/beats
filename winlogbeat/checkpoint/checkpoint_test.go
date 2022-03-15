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

//go:build !integration
// +build !integration

package checkpoint

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func eventually(t *testing.T, predicate func() (bool, error), timeout time.Duration) {
	const minInterval = time.Millisecond * 5
	const maxInterval = time.Millisecond * 500

	checkInterval := timeout / 100
	if checkInterval < minInterval {
		checkInterval = minInterval
	}
	if checkInterval > maxInterval {
		checkInterval = maxInterval
	}
	for deadline, first := time.Now().Add(timeout), true; first || time.Now().Before(deadline); first = false {
		ok, err := predicate()
		if err != nil {
			t.Fatal("predicate failed with error:", err)
			return
		}
		if ok {
			return
		}
		time.Sleep(checkInterval)
	}
	t.Fatal("predicate is not true after", timeout)
}

// Test that a write is triggered when the maximum time period since the last
// write is reached.
func TestWriteTimedFlush(t *testing.T) {
	dir, err := ioutil.TempDir("", "wlb-checkpoint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	}()

	file := filepath.Join(dir, ".winlogbeat.yml")
	if !assert.False(t, fileExists(file), "%s should not exist", file) {
		return
	}

	cp, err := NewCheckpoint(file, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer cp.Shutdown()

	// Send update then wait longer than the flush interval and it should be
	// on disk.
	cp.Persist("App", 1, time.Now(), "")
	eventually(t, func() (bool, error) {
		ps, err := cp.read()
		return ps != nil && len(ps.States) > 0, err
	}, time.Second*15)

	ps, err := cp.read()
	if err != nil {
		t.Fatal("read failed", err)
	}
	if assert.Len(t, ps.States, 1) {
		assert.Equal(t, "App", ps.States[0].Name)
		assert.Equal(t, uint64(1), ps.States[0].RecordNumber)
	}
}

// Test that createDir creates the directory with 0750 permissions.
func TestCreateDir(t *testing.T) {
	dir, err := ioutil.TempDir("", "wlb-checkpoint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	}()

	stateDir := filepath.Join(dir, "state", "dir", "does", "not", "exists")
	file := filepath.Join(stateDir, ".winlogbeat.yml")
	cp := &Checkpoint{file: file}

	if !assert.False(t, fileExists(file), "%s should not exist", file) {
		return
	}
	if err = cp.createDir(); err != nil {
		t.Fatal("createDir", err)
	}
	if !assert.True(t, fileExists(stateDir), "%s should exist", file) {
		return
	}

	// mkdir on Windows does not pass the POSIX mode to the CreateDirectory
	// syscall so doesn't test the mode.
	if runtime.GOOS != "windows" {
		fileInfo, err := os.Stat(stateDir)
		if assert.NoError(t, err) {
			assert.Equal(t, true, fileInfo.IsDir())
			assert.Equal(t, os.FileMode(0o750), fileInfo.Mode().Perm())
		}
	}
}

// Test createDir when the directory already exists to verify that no error is
// returned.
func TestCreateDirAlreadyExists(t *testing.T) {
	dir, err := ioutil.TempDir("", "wlb-checkpoint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	}()

	file := filepath.Join(dir, ".winlogbeat.yml")
	cp := &Checkpoint{file: file}

	if !assert.True(t, fileExists(dir), "%s should exist", file) {
		return
	}
	assert.NoError(t, cp.createDir())
}

// fileExists returns true if the specified file exists.
func fileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}
