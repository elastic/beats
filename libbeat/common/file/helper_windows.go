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
	"os"
)

// SafeFileRotate safely rotates an existing file under path and replaces it with the tempfile
func SafeFileRotate(path, tempfile string) error {
	old := path + ".old"
	var e error

	// In Windows, one cannot rename a file if the destination already exists, at least
	// not with using the os.Rename function that Golang offers.
	// This tries to move the existing file into an old file first and only do the
	// move after that.
	if e = os.Remove(old); e != nil {
		// ignore error in case old doesn't exit yet
	}
	if e = os.Rename(path, old); e != nil {
		// ignore error in case path doesn't exist
	}

	if e = os.Rename(tempfile, path); e != nil {
		return e
	}
	return nil
}
