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
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

var beatName = "testbeat"

func TestMain(m *testing.M) {
	err := logp.DevelopmentSetup()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating logger: %s\n", err)
		os.Exit(1)
	}
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

	os.Exit(m.Run())
}

func TestLockWithDeadPid(t *testing.T) {
	// create old lockfile
	testBeat := beat.Info{Name: mustNewUUID(t), StartTime: time.Now()}
	locker := New(testBeat)
	_, err := locker.createPidfile(8888)
	require.NoError(t, err)

	_, err = locker.fileLock.TryRLock()
	require.NoError(t, err)

	// create new locker
	newLocker := New(testBeat)
	err = newLocker.Lock()
	require.NoError(t, err)
}

func TestLockWithTwoBeats(t *testing.T) {
	testBeat := beat.Info{Name: mustNewUUID(t), StartTime: time.Now()}
	// emulate two beats trying to run from the same data path
	locker := New(testBeat)
	// use pid 1 as another beat
	_, err := locker.createPidfile(1)
	require.NoError(t, err)
	_, err = locker.fileLock.TryRLock()
	require.NoError(t, err)

	// create new locker
	newLocker := New(testBeat)
	err = newLocker.Lock()
	require.Error(t, err)
	t.Logf("Got desired error: %s", err)
}

func TestDoubleLock(t *testing.T) {
	testBeat := beat.Info{Name: mustNewUUID(t), StartTime: time.Now()}
	locker := New(testBeat)
	err := locker.Lock()
	require.NoError(t, err)

	newLocker := New(testBeat)
	err = newLocker.Lock()
	require.Error(t, err)
	t.Logf("Got desired error: %s", err)
}

func TestUnlock(t *testing.T) {
	testBeat := beat.Info{Name: mustNewUUID(t), StartTime: time.Now()}
	locker := New(testBeat)
	err := locker.Lock()
	require.NoError(t, err)

	err = locker.Unlock()
	require.NoError(t, err)
}

func TestRestartWithSamePID(t *testing.T) {
	// create old lockfile
	testBeatName := mustNewUUID(t)
	testBeat := beat.Info{Name: testBeatName, StartTime: time.Now().Add(-time.Second * 20)}
	locker := New(testBeat)
	err := locker.Lock()
	require.NoError(t, err)
	// create new lockfile with the same PID but a newer time
	// create old lockfile
	testNewBeat := beat.Info{Name: testBeatName, StartTime: time.Now()}
	lockerNew := New(testNewBeat)
	err = lockerNew.Lock()
	require.NoError(t, err)
}

func mustNewUUID(t *testing.T) string {
	uuid, err := uuid.NewV4()
	require.NoError(t, err)
	return uuid.String()
}
