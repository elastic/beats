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

package locks

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

func TestMain(m *testing.M) {
	logp.DevelopmentSetup()

	tmp, err := os.MkdirTemp("", "pidfile_test")
	defer os.RemoveAll(tmp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating temp directory: %s\n", err)
		os.Exit(1)
	}

	origDataPath := paths.Paths.Data
	defer func() {
		paths.Paths.Data = origDataPath
	}()
	paths.Paths.Data = tmp

	exit := m.Run()
	// cleanup tmpdir after run, but let the tests set the exit code
	err = os.RemoveAll(tmp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error removing tempdir %s, %s:", tmp, err)
	}

	os.Exit(exit)
}

func TestLocker(t *testing.T) {
	// Setup two beats with same name and data path
	const beatName = "testbeat-testlocker"

	b1 := beat.Info{}
	b1.Beat = beatName

	b2 := beat.Info{}
	b2.Beat = beatName

	// Try to get a lock for the first beat. Expect it to succeed.
	bl1 := New(b1)
	err := bl1.Lock()
	require.NoError(t, err)

	// Try to get a lock for the second beat. Expect it to fail because the
	// first beat already has the lock.
	bl2 := New(b2)
	err = bl2.Lock()
	require.Error(t, err)

}

func TestUnlock(t *testing.T) {
	const beatName = "testbeat-testunlock"

	b1 := beat.Info{}
	b1.Beat = beatName

	b2 := beat.Info{}
	b2.Beat = beatName
	bl2 := New(b2)

	// Try to get a lock for the first beat. Expect it to succeed.
	bl1 := New(b1)
	err := bl1.Lock()
	require.NoError(t, err)

	// now unlock
	err = bl1.Unlock()
	require.NoError(t, err)

	// try with other lockfile
	err = bl2.Lock()
	require.NoError(t, err)

}

func TestUnlockWithRemainingFile(t *testing.T) {
	const beatName = "testbeat-testunlockwithfile"

	b1 := beat.Info{}
	b1.Beat = beatName

	b2 := beat.Info{}
	b2.Beat = beatName
	bl2 := New(b2)

	// Try to get a lock for the first beat. Expect it to succeed.
	bl1 := New(b1)
	err := bl1.Lock()
	require.NoError(t, err)

	// unlock the underlying FD, so we don't remove the file
	err = bl1.fileLock.Unlock()
	require.NoError(t, err)

	// now lock new handle with the same file
	err = bl2.Lock()
	require.NoError(t, err)
}
