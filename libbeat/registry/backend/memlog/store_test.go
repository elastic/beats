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

package memlog

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/registry/backend"
)

func TestLoadVersion1(t *testing.T) {
	dataHome := "testdata/1"

	list, err := ioutil.ReadDir(dataHome)
	if err != nil {
		panic(err)
	}

	cases := list[:0]
	for _, info := range list {
		if info.IsDir() {
			cases = append(cases, info)
		}
	}

	for _, info := range cases {
		name := filepath.Base(info.Name())
		t.Run(name, func(t *testing.T) {
			testLoadVersion1Case(t, filepath.Join(dataHome, info.Name()))
		})
	}
}

func testLoadVersion1Case(t *testing.T, dataPath string) {

	path, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Failed to create temporary test directory: %v", err)
	}
	defer os.RemoveAll(path)

	t.Logf("Test tmp dir: %v", path)

	if err := copyPath(path, dataPath); err != nil {
		t.Fatalf("Failed to copy test file to the temporary directory: %v", err)
	}

	// load expected test results
	raw, err := ioutil.ReadFile(filepath.Join(path, "expected.json"))
	if err != nil {
		t.Fatalf("Failed to load expected.json: %v", err)
	}

	expected := struct {
		Txid     uint64
		Datafile string
		Entries  map[string]interface{}
	}{}
	if err := json.Unmarshal(raw, &expected); err != nil {
		t.Fatalf("Failed to parse expected.json: %v", err)
	}

	// load store:
	store, err := newStore(path, 0660, 4096)
	if err != nil {
		t.Fatalf("Failed to load test store: %v", err)
	}
	defer store.Close()

	disk := store.disk
	disk.removeOldDataFiles()

	// validate store:
	assert.Equal(t, expected.Txid, disk.txid)
	if expected.Datafile != "" {
		assert.Equal(t, filepath.Join(path, expected.Datafile), disk.dataFiles[0].path)
	}

	// check all keys in expected are known and do match stored values:
	func() {
		tx, err := store.Begin(true)
		require.NoError(t, err, "failed to start read transaction")
		defer tx.Close()

		for key, val := range expected.Entries {
			dec, err := tx.Get([]byte(key))
			require.NoError(t, err, "error reading entry")
			assert.NotNil(t, dec, "entry is missing")

			var tmp interface{}
			require.NoError(t, dec.Decode(&tmp), "failed to decode store value")

			assert.Equal(t, val, tmp, "failed when checking key '%s'", key)
		}
	}()

	// check store does not contain any additional keys
	func() {
		tx, err := store.Begin(true)
		require.NoError(t, err, "failed to start read transaction")
		defer tx.Close()

		err = tx.EachKey(false, func(key backend.Key) (bool, error) {
			_, exists := expected.Entries[string(key)]
			if !exists {
				t.Errorf("unexpected key: %s", key)
			}
			return true, nil
		})
		assert.NoError(t, err)
	}()
}

func copyPath(to, from string) error {
	info, err := os.Stat(from)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(to, from)
	}
	if info.Mode().IsRegular() {
		return copyFile(to, from)
	}

	// ignore other file types
	return nil
}

func copyDir(to, from string) error {
	if !isDir(to) {
		info, err := os.Stat(from)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(to, info.Mode()); err != nil {
			return err
		}
	}

	list, err := ioutil.ReadDir(from)
	if err != nil {
		return err
	}

	for _, file := range list {
		name := file.Name()
		err := copyPath(filepath.Join(to, name), filepath.Join(from, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func copyFile(to, from string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(to, os.O_CREATE|os.O_RDWR|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
