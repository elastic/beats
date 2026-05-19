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

//go:build !windows

package readfile

import (
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func createTestFileInfo() file.ExtendedFileInfo {
	return file.ExtendFileInfo(testFileInfo{
		name: "filename",
		size: 42,
		time: time.Now(),
		sys:  &syscall.Stat_t{Dev: 17, Ino: 999, Uid: 0, Gid: 0},
	})
}

func checkFields(t *testing.T, expected, actual mapstr.M) {
	t.Helper()

	require.IsType(t, mapstr.M{}, actual["log"], "expected log to be mapstr.M")
	logMap, _ := actual["log"].(mapstr.M)
	require.IsType(t, mapstr.M{}, logMap["file"], "expected log.file to be mapstr.M")
	fileMap, _ := logMap["file"].(mapstr.M)

	require.Equal(t, "17", fileMap[deviceIDKey])
	delete(fileMap, deviceIDKey)
	require.Equal(t, "999", fileMap[inodeKey])
	delete(fileMap, inodeKey)

	_, hasOwner := fileMap[ownerKey]
	require.False(t, hasOwner)
	_, hasGroup := fileMap[groupKey]
	require.False(t, hasGroup)

	require.Equal(t, expected, actual)
}

// TestCachedMetaSizing verifies that cachedMeta ends up with exactly the right
// number of entries for each combination of optional fields on this platform.
// Because make(mapstr.M, size) pre-allocates exactly `size` slots, matching
// len(cachedMeta) to the formula proves no map growth occurred.
func TestCachedMetaSizing(t *testing.T) {
	fi := createTestFileInfo()
	msg := reader.Message{Content: []byte("line"), Bytes: 4, Fields: mapstr.M{}}

	tests := []struct {
		name         string
		includeOwner bool
		includeGroup bool
		fingerprint  string
		wantLen      int
	}{
		{"base only", false, false, "", 1 + platformFileFields},
		{"with owner", true, false, "", 1 + platformFileFields + 1},
		{"with group", false, true, "", 1 + platformFileFields + 1},
		{"with owner and group", true, true, "", 1 + platformFileFields + 2},
		{"with fingerprint", false, false, "hash", 1 + platformFileFields + 1},
		{"all fields", true, true, "hash", 1 + platformFileFields + 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &FileMetaReader{
				reader:       msgReader([]reader.Message{msg}),
				path:         "test/path",
				fi:           fi,
				includeOwner: tc.includeOwner,
				includeGroup: tc.includeGroup,
				fingerprint:  tc.fingerprint,
			}
			_, err := r.Next()
			require.NoError(t, err)
			require.Len(t, r.cachedMeta, tc.wantLen,
				"cachedMeta entry count should match the pre-allocated size")
		})
	}
}

func checkFieldsWithOwnerGroup(t *testing.T, expected, actual mapstr.M) {
	t.Helper()

	require.IsType(t, mapstr.M{}, actual["log"], "expected log to be mapstr.M")
	logMap, _ := actual["log"].(mapstr.M)
	require.IsType(t, mapstr.M{}, logMap["file"], "expected log.file to be mapstr.M")
	fileMap, _ := logMap["file"].(mapstr.M)

	require.Equal(t, "17", fileMap[deviceIDKey])
	delete(fileMap, deviceIDKey)
	require.Equal(t, "999", fileMap[inodeKey])
	delete(fileMap, inodeKey)

	// macOS uses "wheel" for GID 0; Linux uses "root".
	expectedGroup := "root"
	if runtime.GOOS == "darwin" {
		expectedGroup = "wheel"
	}
	require.Equal(t, "root", fileMap[ownerKey])
	delete(fileMap, ownerKey)
	require.Equal(t, expectedGroup, fileMap[groupKey])
	delete(fileMap, groupKey)

	require.Equal(t, expected, actual)
}
