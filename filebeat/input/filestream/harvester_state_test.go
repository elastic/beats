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

func TestFileStateTable_PublishListsHandle(t *testing.T) {
	tbl := newFileStateTable()

	h := tbl.NewHandle()
	assert.NotNil(t, h, "NewHandle must return a handle")

	// OpenSession pins the handle before the runner publishes it.
	want := nonZeroOSState(t)
	h.PinOSState(want)
	assert.Empty(t, tbl.entries, "NewHandle must not list the handle before Publish")

	h.Publish("id-1", completeDesc("sum-1"))
	assert.Equal(t, "sum-1", h.FingerprintSum(),
		"FingerprintSum must return the completed Sum passed to Publish")

	assert.Same(t, h, tbl.entries["id-1"], "Publish must list the handle under its key")
	assert.Equal(t, want, h.os, "Publish must keep the pinned StateOS")
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

func TestOpenFileState_LongerReplacement(t *testing.T) {
	open := tempFileSource(t, "0123456789")
	other := tempFileSource(t, "0123456789")
	gzipped := other.desc
	gzipped.GZIP = true

	tests := map[string]struct {
		unpinned bool
		desc     loginp.FileDescriptor
		offset   int64
		want     string
	}{
		"a different file larger than the offset":     {desc: other.desc, offset: 9, want: other.newPath},
		"a different file not larger than the offset": {desc: other.desc, offset: 10},
		"the open file itself":                        {desc: open.desc},
		"a GZIP file":                                 {desc: gzipped},
		"an unknown open file identity":               {unpinned: true, desc: other.desc},
		"no scanner descriptor":                       {},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			h := publishHandle(newFileStateTable(), "id", tc.desc)
			if !tc.unpinned {
				h.PinOSState(open.desc.Info.GetOSState())
			}
			assert.Equal(t, tc.want, h.LongerReplacement(tc.offset), "LongerReplacement must return the scanned path or none")
		})
	}
}

func TestFileStateTable_UpdateDescriptorUpdatesIfPresent(t *testing.T) {
	tbl := newFileStateTable()

	tbl.UpdateDescriptor("absent", completeDesc("sum"))
	assert.NotContains(t, tbl.entries, "absent", "UpdateDescriptor must not insert an entry for an absent key")

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

	assert.NotContains(t, tbl.entries, "old-id", "old key must no longer resolve after Publish")
	assert.Same(t, h, tbl.entries["new-id"], "new key must resolve to the moved handle after Publish")
	assert.Equal(t, pinned, h.os, "Publish must preserve the pinned StateOS")
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
	second := publishHandle(tbl, "fingerprint::short", growingDesc("deadbeef"))

	first.Publish("fingerprint::full", completeDesc("full-sum"))

	assert.Same(t, first, tbl.entries["fingerprint::full"], "a displaced handle must still move to its new key")
	assert.Equal(t, "full-sum", first.FingerprintSum(),
		"the moved handle must expose the new identity's fingerprint")
	assert.Same(t, second, tbl.entries["fingerprint::short"], "the second handle must keep the old key")

	tbl.Deregister(second)
	tbl.Deregister(first)
	assert.Empty(t, tbl.entries, "both handles must deregister cleanly")
}

func TestFileStateTable_DeregisterCompareAndDelete(t *testing.T) {
	t.Run("removes its own entry", func(t *testing.T) {
		tbl := newFileStateTable()
		h := publishHandle(tbl, "id-1", completeDesc("sum"))

		tbl.Deregister(h)
		assert.NotContains(t, tbl.entries, "id-1", "Deregister must remove the handle's own entry")
	})

	t.Run("displaced handle Deregister no-ops under Restart overlap", func(t *testing.T) {
		tbl := newFileStateTable()

		// During a restart, the new harvester can publish before the old one closes.
		old := publishHandle(tbl, "id-1", completeDesc("old"))
		newer := publishHandle(tbl, "id-1", completeDesc("new"))

		// Removing the old handle must preserve the newer entry.
		tbl.Deregister(old)
		assert.Same(t, newer, tbl.entries["id-1"], "the newer handle must survive the displaced handle's Deregister")

		// The newer handle can still deregister itself.
		tbl.Deregister(newer)
		assert.NotContains(t, tbl.entries, "id-1", "the newer handle must be able to deregister itself")
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

	h.PinOSState(nonZeroOSState(t))
	assert.Empty(t, h.FingerprintSum(), "FingerprintSum on a nil handle must be empty")
	assert.Empty(t, h.LongerReplacement(0), "LongerReplacement on a nil handle must report no replacement")

	// A handle obtained from a real table must tolerate a nil-table Deregister path too.
	realTable := newFileStateTable()
	realHandle := publishHandle(realTable, "id", completeDesc("sum"))
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
		var current atomic.Pointer[openFileState]

		wg.Go(func() {
			for range ops {
				h := publishHandle(tbl, key, growingDesc("dead"))
				current.Store(h)
				h.PinOSState(pinned)
				_ = h.FingerprintSum()
				tbl.Deregister(h)
			}
		})

		wg.Go(func() {
			for range ops {
				tbl.UpdateDescriptor(key, completeDesc("sum"))
				_ = current.Load().LongerReplacement(0)
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
