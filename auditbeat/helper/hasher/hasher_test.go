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

package hasher

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasher(t *testing.T) {
	dir, err := ioutil.TempDir("", "auditbeat-hasher-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	file := filepath.Join(dir, "exe")
	if err = ioutil.WriteFile(file, []byte("test exe\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	config := Config{
		HashTypes:           []HashType{SHA1, MD5},
		MaxFileSize:         "100 MiB",
		MaxFileSizeBytes:    100 * 1024 * 1024,
		ScanRatePerSec:      "50 MiB",
		ScanRateBytesPerSec: 50 * 1024 * 1024,
	}
	hasher, err := NewFileHasher(config, nil)
	if err != nil {
		t.Fatal(err)
	}

	hashes, err := hasher.HashFile(file)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, hashes, 2)
	assert.Equal(t, "44a36f2cd27e56794cd405ad8d44e82dba4c54fa", hashes["sha1"].String())
	assert.Equal(t, "1d7572082f6b0d18a393d618285d7100", hashes["md5"].String())
}

func TestHasherLimits(t *testing.T) {
	dir, err := ioutil.TempDir("", "auditbeat-hasher-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	file := filepath.Join(dir, "exe")
	if err = ioutil.WriteFile(file, []byte("test exe\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	configZeroSize := Config{
		HashTypes:           []HashType{SHA1},
		MaxFileSize:         "0 MiB",
		MaxFileSizeBytes:    0,
		ScanRatePerSec:      "0 MiB",
		ScanRateBytesPerSec: 0,
	}
	hasher, err := NewFileHasher(configZeroSize, nil)
	if err != nil {
		t.Fatal(err)
	}

	hashes, err := hasher.HashFile(file)
	assert.Empty(t, hashes)
	assert.Error(t, err)
	assert.IsType(t, FileTooLargeError{}, cause(err))
}

func cause(err error) error {
	type unwrapper interface {
		Unwrap() error
	}

	for err != nil {
		w, ok := err.(unwrapper)
		if !ok {
			break
		}
		err = w.Unwrap()
	}
	return err
}
