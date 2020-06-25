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
	"os"
	"syscall"
)

// isTxIDLessEqual compares two IDs by checking that their distance is < 2^63.
// It always returns true if
//  - a == b
//  - a < b (mod 2^63)
//  - b > a after an integer rollover that is still within the distance of <2^63-1
func isTxIDLessEqual(a, b uint64) bool {
	return int64(a-b) <= 0
}

func isRetryErr(err error) bool {
	return err == syscall.EINTR || err == syscall.EAGAIN
}

// trySyncPath provides a best-effort fsync on path (directory). The fsync is required by some
// filesystems, so to update the parents directory metadata to actually
// contain the new file being rotated in.
func trySyncPath(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // ignore error, sync on dir must not be necessarily supported by the FS
	}
	defer f.Close()
	syncFile(f)
}
