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

package filestream

import (
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/file"
)

// tempFileInfo returns the stat of a fresh temp file. Distinct calls yield
// distinct OS identities on every platform.
func tempFileInfo(t *testing.T) file.ExtendedFileInfo {
	t.Helper()
	fi, err := os.Stat(writeTempFile(t, "x"))
	require.NoError(t, err, "stat temp file")
	return file.ExtendFileInfo(fi)
}

// nonZeroOSState returns the non-zero StateOS of a fresh temp file.
func nonZeroOSState(t *testing.T) file.StateOS {
	t.Helper()
	st := tempFileInfo(t).GetOSState()
	require.NotEqual(t, file.StateOS{}, st, "stat must produce a non-zero StateOS")
	return st
}

func TestFileStateTable_RegisterPinLookup(t *testing.T) {
	tbl := newFileStateTable()

	// No entry yet.
	_, ok := tbl.LookupOSState("id-1")
	assert.False(t, ok, "LookupOSState must fail before any Register")

	h := tbl.Register("id-1")
	assert.NotNil(t, h, "Register must return a handle")

	// Registered but not pinned: no OS state yet.
	_, ok = tbl.LookupOSState("id-1")
	assert.False(t, ok, "LookupOSState must fail on an unpinned entry")

	want := nonZeroOSState(t)
	h.PinOSState(want)
	got, ok := tbl.LookupOSState("id-1")
	assert.True(t, ok, "LookupOSState must succeed on a pinned, non-zero entry")
	assert.Equal(t, want, got, "LookupOSState must return the pinned StateOS")
}

func TestFileStateTable_LookupZeroStateOSReadsAsNoPin(t *testing.T) {
	tbl := newFileStateTable()
	h := tbl.Register("id-1")

	// A zero StateOS (e.g. Windows loadFileId failing) must read as "no pin".
	h.PinOSState(file.StateOS{})
	_, ok := tbl.LookupOSState("id-1")
	assert.False(t, ok, "a zero pinned StateOS must read as no pin")
}

func TestFileStateTable_DeregisterCompareAndDelete(t *testing.T) {
	t.Run("removes its own entry", func(t *testing.T) {
		tbl := newFileStateTable()
		h := tbl.Register("id-1")
		h.PinOSState(nonZeroOSState(t))

		tbl.Deregister(h)
		_, ok := tbl.LookupOSState("id-1")
		assert.False(t, ok, "Deregister must remove the handle's own entry")
	})

	t.Run("displaced handle Deregister no-ops under Restart overlap", func(t *testing.T) {
		tbl := newFileStateTable()

		// During a restart, the new harvester can register before the old one closes.
		old := tbl.Register("id-1")
		newer := tbl.Register("id-1")
		newerPin := nonZeroOSState(t)
		newer.PinOSState(newerPin)

		// Removing the old handle must preserve the newer entry.
		tbl.Deregister(old)
		got, ok := tbl.LookupOSState("id-1")
		assert.True(t, ok, "the newer handle must survive the displaced handle's Deregister")
		assert.Equal(t, newerPin, got, "the surviving entry must be the newer handle's")

		// The newer handle can still deregister itself.
		tbl.Deregister(newer)
		_, ok = tbl.LookupOSState("id-1")
		assert.False(t, ok, "the newer handle must be able to deregister itself")
	})
}

func TestFileStateTable_NilSafety(t *testing.T) {
	var tbl *fileStateTable
	var h *openFileState

	// None of these must panic on a nil table / nil handle.
	assert.Nil(t, tbl.Register("id"),
		"Register on a nil table must return nil")
	tbl.Deregister(h)
	_, ok := tbl.LookupOSState("id")
	assert.False(t, ok, "LookupOSState on a nil table must report no pin")

	h.PinOSState(nonZeroOSState(t))
	// A handle obtained from a real table must tolerate a nil-table Deregister path too.
	realTable := newFileStateTable()
	realHandle := realTable.Register("id")
	tbl.Deregister(realHandle) // nil table, real handle: still a no-op, no panic
}

// TestFileStateTable_ConcurrentAccess runs readers and writers concurrently.
func TestFileStateTable_ConcurrentAccess(t *testing.T) {
	tbl := newFileStateTable()
	pinned := nonZeroOSState(t)

	const workers = 16
	const ops = 200
	var wg sync.WaitGroup

	for i := range workers {
		key := "id-" + strconv.Itoa(i)
		// Harvester goroutine: Register, Pin, Deregister.
		wg.Go(func() {
			for range ops {
				h := tbl.Register(key)
				h.PinOSState(pinned)
				tbl.Deregister(h)
			}
		})

		// Scanner goroutine: LookupOSState.
		wg.Go(func() {
			for range ops {
				_, _ = tbl.LookupOSState(key)
			}
		})
	}

	wg.Wait()
}
