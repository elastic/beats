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
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func createTestFileInfo() file.ExtendedFileInfo {
	return file.ExtendFileInfo(testFileInfo{
		name: "filename",
		size: 42,
		time: time.Now(),
		sys:  &syscall.Stat_t{Dev: 17, Ino: 999},
	})
}

func checkFields(t *testing.T, expected, actual mapstr.M) {
	t.Helper()

	dev, err := actual.GetValue(deviceIDKey)
	require.NoError(t, err)
	require.Equal(t, "17", dev)
	err = actual.Delete(deviceIDKey)
	require.NoError(t, err)

	inode, err := actual.GetValue(inodeKey)
	require.NoError(t, err)
	require.Equal(t, "999", inode)
	err = actual.Delete(inodeKey)
	require.NoError(t, err)

	require.Equal(t, expected, actual)
}
