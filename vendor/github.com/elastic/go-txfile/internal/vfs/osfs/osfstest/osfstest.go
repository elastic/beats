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

package osfstest

import (
	"io/ioutil"
	"os"
	"path"
)

type testing interface {
	Fatal(v ...interface{})
}

// SetupPath creates a temporary directory and a test file, based on the passed
// file name.  The path to the temporary test file and a teardown function for
// deleting the temporary directory are returned.
// On failure the Fatal method of t will be executed.
func SetupPath(t testing, file string) (fileName string, teardown func()) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	if file == "" {
		file = "test.dat"
	}
	return path.Join(dir, file), func() {
		os.RemoveAll(dir)
	}
}
