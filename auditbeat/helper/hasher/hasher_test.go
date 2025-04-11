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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasher(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "exe")
	if err := os.WriteFile(file, []byte("test exe\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	config := Config{
		HashTypes:           []HashType{SHA256, SHA512},
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
	assert.Equal(t, "c2bf6d47d4b367498fba613e6b7b33798b713f4909dfdf4f2b8a919c5440d36e", hashes["sha256"].String())
	assert.Equal(t, "6908dddec81668e11b7c1d65f27f5b26e85bd02847980e476e951cf9d740dd154f328e76f471854c2cb21a0e87a9b9f65c90b51a23baa988a45d1f91d57f88ce", hashes["sha512"].String())
}

func TestHasherLimits(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "exe")
	if err := os.WriteFile(file, []byte("test exe\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	configZeroSize := Config{
		HashTypes:           []HashType{SHA256},
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
	assert.True(t, errors.As(err, &FileTooLargeError{}))
}
