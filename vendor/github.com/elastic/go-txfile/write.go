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

package txfile

import (
	"io"
	"sort"
	"sync"

	"github.com/elastic/go-txfile/internal/vfs"
)

type writer struct {
	target   writable
	pageSize uint

	mux        sync.Mutex
	cond       *sync.Cond
	done       bool
	scheduled  []writeMsg
	scheduled0 [64]writeMsg
	fsync      []syncMsg
	fsync0     [8]syncMsg

	syncMode SyncMode

	pending   int // number of scheduled writes since last sync
	published int // number of writes executed since last sync
}

type writeMsg struct {
	sync  *txWriteSync
	id    PageID
	buf   []byte
	fsync bool
}

type syncMsg struct {
	sync  *txWriteSync
	count int // number of pages to process, before fsyncing
	flags syncFlag
}

type txWriteSync struct {
	err reason
	wg  sync.WaitGroup
}

type writable interface {
	io.WriterAt
	Sync(vfs.SyncFlag) error
}

// command as is consumer by the writers run loop
type command struct {
	n         int          // number of buffered write message to be consumed
	fsync     *txWriteSync // set if fsync is to be executed after writing all messages
	syncFlags syncFlag     // additional fsync flags
}

type syncFlag uint8

const (
	// On IO error, the writer will ignore any write/sync attempts, but return
	// the first error encountered.
	// Passing the syncResetErr notifies the writer that the current transaction
	// is about to fail and all subsequent writes will belong to a new
	// transaction. So to not stall any writes/operations forever, the writer
	// will attempt write/sync any future requests, by resetting the internal
	// error state to 'no error'.
	syncResetErr syncFlag = 1 << iota

	// syncDataOnly tells the writer that we don't care about metadata updates like
	// access/modification timestamps (given the file size didn't change).
	// Some filesystems profit from data only syncs, as meta data or journals
	// don't need to be flushed, reducing the overall amount of on disk IO ops.
	syncDataOnly
)

func (w *writer) Init(target writable, pageSize uint, syncMode SyncMode) {
	if syncMode == SyncDefault {
		syncMode = SyncData
	}

	w.target = target
	w.syncMode = syncMode
	w.pageSize = pageSize
	w.cond = sync.NewCond(&w.mux)
	w.scheduled = w.scheduled0[:0]
	w.fsync = w.fsync[:0]
}

func (w *writer) Stop() {
	w.mux.Lock()
	w.done = true
	w.mux.Unlock()
	w.cond.Signal()
}

func (w *writer) Schedule(sync *txWriteSync, id PageID, buf []byte) {
	sync.Retain()
	traceln("schedule write")

	w.mux.Lock()
	defer w.mux.Unlock()
	w.scheduled = append(w.scheduled, writeMsg{
		sync: sync,
		id:   id,
		buf:  buf,
	})
	w.pending++

	w.cond.Signal()
}

func (w *writer) Sync(sync *txWriteSync, flags syncFlag) {
	sync.Retain()
	traceln("schedule sync")

	w.mux.Lock()
	defer w.mux.Unlock()
	w.fsync = append(w.fsync, syncMsg{
		sync:  sync,
		count: w.pending,
		flags: flags,
	})
	w.pending = 0

	w.cond.Signal()
}

func (w *writer) Run() (bool, reason) {
	var (
		err  reason
		done bool
		cmd  command
		buf  [1024]writeMsg
	)

	for {
		cmd, done = w.nextCommand(buf[:])
		if done {
			return done, nil
		}

		traceln("writer message: ", cmd.n, cmd.fsync != nil, done)

		// TODO: use vector IO if possible (linux: pwritev)
		msgs := buf[:cmd.n]
		sort.Slice(msgs, func(i, j int) bool {
			return msgs[i].id < msgs[j].id
		})
		for _, msg := range msgs {
			const op = "txfile/write-page"

			if err == nil {
				// execute actual write on the page it's file offset:
				off := uint64(msg.id) * uint64(w.pageSize)
				tracef("write at(id=%v, off=%v, len=%v)\n", msg.id, off, len(msg.buf))

				err = writeAt(op, w.target, msg.buf, int64(off))
			}

			msg.sync.err = err
			msg.sync.Release()
		}

		// execute pending fsync:
		if fsync := cmd.fsync; fsync != nil {
			if err == nil {
				err = w.execSync(cmd)
			}
			fsync.err = err

			traceln("done fsync")
			fsync.Release()

			resetErr := cmd.syncFlags.Test(syncResetErr)
			if resetErr {
				err = nil
			}
		}
	}
}

func (w *writer) execSync(cmd command) reason {
	const op = "txfile/write-sync"

	syncFlag := vfs.SyncAll
	switch w.syncMode {
	case SyncNone:
		return nil

	case SyncData:
		if cmd.syncFlags.Test(syncDataOnly) {
			syncFlag = vfs.SyncDataOnly
		}
	}

	if err := w.target.Sync(syncFlag); err != nil {
		return errOp(op).causedBy(err)
	}

	return nil
}

func (w *writer) nextCommand(buf []writeMsg) (command, bool) {
	w.mux.Lock()
	defer w.mux.Unlock()

	traceln("async writer: wait next command")
	defer traceln("async writer: received next command")

	for {
		if w.done {
			return command{}, true
		}

		max := len(w.scheduled)
		if max == 0 && len(w.fsync) == 0 { // no messages
			w.cond.Wait()
			continue
		}

		if l := len(buf); l < max {
			max = l
		}

		// Check if we need to fsync and adjust `max` number of pages of required.
		var sync *txWriteSync
		var syncFlags syncFlag
		traceln("check fsync: ", len(w.fsync))

		if len(w.fsync) > 0 {
			msg := w.fsync[0]

			// number of outstanding scheduled writes before fsync
			outstanding := msg.count - w.published
			traceln("outstanding:", outstanding)

			if outstanding <= max { // -> fsync
				max, sync, syncFlags = outstanding, msg.sync, msg.flags

				// advance fsync state
				w.fsync[0] = syncMsg{} // clear entry, so to potentially clean references from w.fsync0
				w.fsync = w.fsync[1:]
				if len(w.fsync) == 0 {
					w.fsync = w.fsync0[:0]
				}
			}
		}

		// return buffers to be processed
		var n int
		scheduled := w.scheduled[:max]
		if len(scheduled) > 0 {
			n = copy(buf, scheduled)
			w.scheduled = w.scheduled[n:]
			if len(w.scheduled) == 0 {
				w.scheduled = w.scheduled0[:0]
			}
		}

		if sync == nil {
			w.published += n
		} else {
			w.published = 0
		}

		return command{n: n, fsync: sync, syncFlags: syncFlags}, false
	}
}

func newTxWriteSync() *txWriteSync {
	return &txWriteSync{}
}

func (s *txWriteSync) Retain() {
	s.wg.Add(1)
}

func (s *txWriteSync) Release() {
	s.wg.Done()
}

func (s *txWriteSync) Wait() reason {
	s.wg.Wait()
	return s.err
}

func writeAt(op string, out io.WriterAt, buf []byte, off int64) reason {
	for len(buf) > 0 {
		n, err := out.WriteAt(buf, off)
		if err != nil {
			return errOp(op).causedBy(err).
				reportf("writing %v bytes to off=%v failed", len(buf), off)
		}

		off += int64(n)
		buf = buf[n:]
	}
	return nil
}

func (f syncFlag) Test(other syncFlag) bool {
	return (f & other) == other
}
