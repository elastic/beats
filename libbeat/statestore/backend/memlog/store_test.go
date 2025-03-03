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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestRecoverFromCorruption(t *testing.T) {
	path := t.TempDir()
	logp.DevelopmentSetup()

	if err := copyPath(path, "testdata/1/logfile_incomplete/"); err != nil {
		t.Fatalf("Failed to copy test file to the temporary directory: %v", err)
	}

	store, err := openStore(logp.NewLogger("test"), path, 0660, 4096, false, func(_ uint64) bool {
		return false
	})
	require.NoError(t, err, "openStore must succeed")
	require.True(t, store.disk.logInvalid, "expecting the log file to be invalid")

	err = store.logOperation(&opSet{K: "key", V: mapstr.M{
		"field": 42,
	}})
	require.NoError(t, err, "logOperation must succeed")
	require.False(t, store.disk.logInvalid, "log file must be valid")
	require.FileExistsf(t, filepath.Join(path, "7.json"), "expecting the checkpoint file to have been created")

	file, err := os.Stat(filepath.Join(path, "log.json"))
	require.NoError(t, err, "Stat on the log file must succeed")
	require.Equal(t, int64(0), file.Size(), "expecting the log file to be truncated")
}
