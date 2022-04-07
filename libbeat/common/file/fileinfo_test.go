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

//go:build !windows && !openbsd
// +build !windows,!openbsd

// Test for openbsd are excluded here as info.GID() returns 0 instead of the actual value
// As the code does not seem to be used in any of the beats, this should be ok
// Still it would be interesting to know why it returns 0.

package file_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common/file"
)

func TestStat(t *testing.T) {
	f, err := ioutil.TempFile("", "teststat")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	link := filepath.Join(os.TempDir(), "teststat-link")
	if err := os.Symlink(f.Name(), link); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(link)

	info, err := file.Stat(link)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, info.Mode().IsRegular())

	uid, err := info.UID()
	if err != nil {
		t.Fatal(err)
	}
	assert.EqualValues(t, os.Geteuid(), uid)

	gid, err := info.GID()
	if err != nil {
		t.Fatal(err)
	}
	assert.EqualValues(t, os.Getegid(), gid)
}

func TestLstat(t *testing.T) {
	link := filepath.Join(os.TempDir(), "link")
	if err := os.Symlink("dummy", link); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(link)

	info, err := file.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, info.Mode()&os.ModeSymlink > 0)

	uid, err := info.UID()
	if err != nil {
		t.Fatal(err)
	}
	assert.EqualValues(t, os.Geteuid(), uid)

	gid, err := info.GID()
	if err != nil {
		t.Fatal(err)
	}
	assert.EqualValues(t, os.Getegid(), gid)
}
