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
	"os"
	"testing"

	"github.com/hectane/go-acl"
	"github.com/stretchr/testify/assert"
)

// TestFileInfoPermissions tests obtaining metadata of a file
// when we don't have permissions to open the file for reading.
// This prevents us to get the file owner of a file unless we use
// a method that doesn't need to open the file for reading.
// (GetNamedSecurityInfo vs CreateFile+GetSecurityInfo)
func TestFileInfoPermissions(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "metadata")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	name := f.Name()

	makeFileNonReadable(t, f.Name())
	info, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := NewMetadata(name, info)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	t.Log(meta.Owner)
	assert.NotEqual(t, "", meta.Owner)
}

func makeFileNonReadable(t testing.TB, path string) {
	if err := acl.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
}
