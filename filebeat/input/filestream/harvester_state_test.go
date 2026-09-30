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
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/common/file"
)

func completeDesc(sum string) loginp.FileDescriptor {
	return loginp.FileDescriptor{Fingerprint: completeFP(sum)}
}

func growingDesc(raw string) loginp.FileDescriptor {
	return loginp.FileDescriptor{Fingerprint: loginp.FingerprintID{Raw: raw}}
}

// tempFileInfo returns the stat of a fresh temp file. Distinct calls yield
// distinct OS identities on every platform.
func tempFileInfo(t *testing.T) file.ExtendedFileInfo {
	t.Helper()
	return tempFileSource(t, "x").desc.Info
}

// nonZeroOSState returns the non-zero StateOS of a fresh temp file.
func nonZeroOSState(t *testing.T) file.StateOS {
	t.Helper()
	st := tempFileInfo(t).GetOSState()
	require.NotEqual(t, file.StateOS{}, st, "stat must produce a non-zero StateOS")
	return st
}

func publishHandle(tbl *fileStateTable, name string, desc loginp.FileDescriptor) *openFileState {
	h := tbl.NewHandle()
	h.Publish(name, desc)
	return h
}

func TestFileStateTable_PublishPinLookup(t *testing.T) {
	tbl := newFileStateTable()

	// No entry yet.
	_, ok := tbl.LookupOSState("id-1")
	assert.False(t, ok, "LookupOSState must fail before any Publish")

	h := tbl.NewHandle()
	assert.NotNil(t, h, "NewHandle must return a handle")

	// OpenSession pins the handle before the runner publishes it.
	want := nonZeroOSState(t)
	h.PinOSState(want)
	assert.Empty(t, tbl.entries, "NewHandle must not list the handle before Publish")

	h.Publish("id-1", completeDesc("sum-1"))
	assert.Equal(t, "sum-1", h.FingerprintSum(),
		"FingerprintSum must return the completed Sum passed to Publish")

	got, ok := tbl.LookupOSState("id-1")
	assert.True(t, ok, "LookupOSState must succeed on a pinned, non-zero entry")
	assert.Equal(t, want, got, "LookupOSState must return the pinned StateOS")
}

func TestFileStateTable_LookupUnpinnedReadsAsNoPin(t *testing.T) {
	tbl := newFileStateTable()
	h := publishHandle(tbl, "id-1", completeDesc("sum-1"))

	_, ok := tbl.LookupOSState("id-1")
	assert.False(t, ok, "LookupOSState must fail on an unpinned entry")

	// A zero StateOS (e.g. Windows loadFileId failing) must read as "no pin".
	h.PinOSState(file.StateOS{})
	_, ok = tbl.LookupOSState("id-1")
	assert.False(t, ok, "a zero pinned StateOS must read as no pin")
}

func TestFileStateTable_FingerprintSum(t *testing.T) {
	tbl := newFileStateTable()

	complete := publishHandle(tbl, "complete", completeDesc("the-sum"))
	assert.Equal(t, "the-sum", complete.FingerprintSum(),
		"a completed fingerprint must expose its Sum")

	growing := publishHandle(tbl, "growing", growingDesc("deadbeef"))
	assert.Empty(t, growing.FingerprintSum(),
		"an incomplete fingerprint must expose no Sum")
}

func TestFileStateTable_UpdateDescriptorUpdatesIfPresent(t *testing.T) {
	tbl := newFileStateTable()

	tbl.UpdateDescriptor("absent", completeDesc("sum"))
	_, ok := tbl.LookupOSState("absent")
	assert.False(t, ok, "UpdateDescriptor must not insert an entry for an absent key")

	h := publishHandle(tbl, "id-1", growingDesc("deadbeef"))
	assert.Empty(t, h.FingerprintSum(), "handle must start below threshold with no Sum")

	tbl.UpdateDescriptor("id-1", completeDesc("final-sum"))
	assert.Equal(t, "final-sum", h.FingerprintSum(),
		"UpdateDescriptor must make the completed Sum visible on the handle")
}

func TestFileStateTable_PublishMovesHandle(t *testing.T) {
	tbl := newFileStateTable()
	h := publishHandle(tbl, "old-id", growingDesc("dead"))
	pinned := nonZeroOSState(t)
	h.PinOSState(pinned)

	h.Publish("new-id", completeDesc("sum-1"))

	_, ok := tbl.LookupOSState("old-id")
	assert.False(t, ok, "old key must no longer resolve after Publish")

	got, ok := tbl.LookupOSState("new-id")
	assert.True(t, ok, "new key must resolve to the moved handle after Publish")
	assert.Equal(t, pinned, got, "Publish must preserve the pinned StateOS")
	assert.Equal(t, "sum-1", h.FingerprintSum(), "Publish must store the new identity's descriptor")

	tbl.UpdateDescriptor("new-id", completeDesc("sum-2"))
	assert.Equal(t, "sum-2", h.FingerprintSum(),
		"UpdateDescriptor under the new key must reach the moved handle")
	tbl.UpdateDescriptor("old-id", completeDesc("ignored"))
	assert.Equal(t, "sum-2", h.FingerprintSum(),
		"UpdateDescriptor under the stale old key must not reach the handle")
}

// TestFileStateTable_PublishDisplacedHandle moves a handle whose old key a
// newer handle took. The newer handle keeps the old key and the moved handle
// must be listed under its new key.
func TestFileStateTable_PublishDisplacedHandle(t *testing.T) {
	tbl := newFileStateTable()

	first := publishHandle(tbl, "fingerprint::short", growingDesc("deadbeef"))
	firstPin := nonZeroOSState(t)
	first.PinOSState(firstPin)

	second := publishHandle(tbl, "fingerprint::short", growingDesc("deadbeef"))
	secondPin := nonZeroOSState(t)
	second.PinOSState(secondPin)

	first.Publish("fingerprint::full", completeDesc("full-sum"))

	got, ok := tbl.LookupOSState("fingerprint::full")
	require.True(t, ok, "a displaced handle must still move to its new key")
	assert.Equal(t, firstPin, got, "the new key must resolve to the first handle's identity")
	assert.Equal(t, "full-sum", first.FingerprintSum(),
		"the moved handle must expose the new identity's fingerprint")

	got, ok = tbl.LookupOSState("fingerprint::short")
	assert.True(t, ok, "the second handle must keep the old key")
	assert.Equal(t, secondPin, got, "the old key must resolve to the second handle's identity")

	tbl.Deregister(second)
	tbl.Deregister(first)
	assert.Empty(t, tbl.entries, "both handles must deregister cleanly")
}

func TestFileStateTable_DeregisterCompareAndDelete(t *testing.T) {
	t.Run("removes its own entry", func(t *testing.T) {
		tbl := newFileStateTable()
		h := publishHandle(tbl, "id-1", completeDesc("sum"))
		h.PinOSState(nonZeroOSState(t))

		tbl.Deregister(h)
		_, ok := tbl.LookupOSState("id-1")
		assert.False(t, ok, "Deregister must remove the handle's own entry")
	})

	t.Run("displaced handle Deregister no-ops under Restart overlap", func(t *testing.T) {
		tbl := newFileStateTable()

		// During a restart, the new harvester can publish before the old one closes.
		old := publishHandle(tbl, "id-1", completeDesc("old"))
		newer := publishHandle(tbl, "id-1", completeDesc("new"))
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
	assert.Nil(t, tbl.NewHandle(), "NewHandle on a nil table must return nil")
	tbl.UpdateDescriptor("id", completeDesc("sum"))
	h.Publish("b", completeDesc("sum"))
	tbl.Deregister(h)
	_, ok := tbl.LookupOSState("id")
	assert.False(t, ok, "LookupOSState on a nil table must report no pin")

	h.PinOSState(nonZeroOSState(t))
	assert.Empty(t, h.FingerprintSum(), "FingerprintSum on a nil handle must be empty")

	// A handle obtained from a real table must tolerate a nil-table Deregister path too.
	realTable := newFileStateTable()
	realHandle := publishHandle(realTable, "id", completeDesc("sum"))
	tbl.Deregister(realHandle) // nil table, real handle: still a no-op, no panic
}

func TestFileStateTable_PinnedDescriptor(t *testing.T) {
	tbl := newFileStateTable()
	pin := nonZeroOSState(t)
	descAt := func(path, sum string) loginp.FileDescriptor {
		return loginp.FileDescriptor{Filename: path, Fingerprint: completeFP(sum)}
	}

	_, _, ok := tbl.PinnedDescriptor("/logs/a.log")
	assert.False(t, ok, "PinnedDescriptor must fail before any Publish")

	h := publishHandle(tbl, "id-1", descAt("/logs/a.log", "sum-1"))
	_, _, ok = tbl.PinnedDescriptor("/logs/a.log")
	assert.False(t, ok, "PinnedDescriptor must fail before PinOSState")

	// A zero StateOS is not a usable identity: still no candidate.
	h.PinOSState(file.StateOS{})
	_, _, ok = tbl.PinnedDescriptor("/logs/a.log")
	assert.False(t, ok, "a zero pin must not make the handle a candidate")

	h.PinOSState(pin)
	got, gotPin, ok := tbl.PinnedDescriptor("/logs/a.log")
	assert.True(t, ok, "PinnedDescriptor must resolve the pinned handle at its path")
	assert.Equal(t, "sum-1", got.Fingerprint.Sum, "PinnedDescriptor must return the handle's descriptor")
	assert.Equal(t, pin, gotPin, "PinnedDescriptor must return the handle's pin")

	// The scanner reports the file at a new path (rename): the index follows.
	tbl.UpdateDescriptor("id-1", descAt("/logs/b.log", "sum-1"))
	_, _, ok = tbl.PinnedDescriptor("/logs/a.log")
	assert.False(t, ok, "the old path must no longer resolve after the descriptor moved")
	got, _, ok = tbl.PinnedDescriptor("/logs/b.log")
	assert.True(t, ok, "the new path must resolve after the descriptor moved")
	assert.Equal(t, "sum-1", got.Fingerprint.Sum, "the moved entry must keep its descriptor")

	tbl.Deregister(h)
	_, _, ok = tbl.PinnedDescriptor("/logs/b.log")
	assert.False(t, ok, "PinnedDescriptor must fail after Deregister clears the index")
}

func TestFileStateTable_PinnedDescriptorDisplacedHandle(t *testing.T) {
	tbl := newFileStateTable()
	desc := loginp.FileDescriptor{Filename: "/logs/a.log", Fingerprint: completeFP("old")}

	// A replacement harvester takes ownership of both indexes before the old
	// harvester closes.
	old := publishHandle(tbl, "id-old", desc)
	old.PinOSState(nonZeroOSState(t))
	desc.Fingerprint = completeFP("new")
	newer := publishHandle(tbl, "id-new", desc)
	newer.PinOSState(nonZeroOSState(t))

	tbl.Deregister(old)
	got, _, ok := tbl.PinnedDescriptor("/logs/a.log")
	assert.True(t, ok, "the newer handle must keep the path index after the displaced one deregisters")
	assert.Equal(t, "new", got.Fingerprint.Sum, "the surviving index entry must be the newer handle's")

	tbl.Deregister(newer)
	_, _, ok = tbl.PinnedDescriptor("/logs/a.log")
	assert.False(t, ok, "the newer handle's Deregister must clear the path index")
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
		var current atomic.Pointer[openFileState]

		wg.Go(func() {
			for range ops {
				h := publishHandle(tbl, key, loginp.FileDescriptor{Filename: "/logs/" + key, Fingerprint: loginp.FingerprintID{Raw: "dead"}})
				current.Store(h)
				h.PinOSState(pinned)
				_ = h.FingerprintSum()
				tbl.Deregister(h)
			}
		})

		wg.Go(func() {
			for range ops {
				tbl.UpdateDescriptor(key, loginp.FileDescriptor{Filename: "/logs/" + key + ".1", Fingerprint: completeFP("sum")})
				_, _ = tbl.LookupOSState(key)
				_, _, _ = tbl.PinnedDescriptor("/logs/" + key)
				_, _, _ = tbl.PinnedDescriptor("/logs/" + key + ".1")
			}
		})

		wg.Go(func() {
			for range ops {
				h := current.Load()
				h.Publish(key+"-moved", growingDesc("dead"))
				h.Publish(key, growingDesc("dead"))
			}
		})
	}

	wg.Wait()
}
