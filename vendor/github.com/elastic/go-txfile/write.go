package txfile

import (
	"io"
	"sort"
	"sync"
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
}

type txWriteSync struct {
	err error
	wg  sync.WaitGroup
}

type writable interface {
	io.WriterAt
	Sync() error
}

func (w *writer) Init(target writable, pageSize uint) {
	w.target = target
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

func (w *writer) Sync(sync *txWriteSync) {
	sync.Retain()
	traceln("schedule sync")

	w.mux.Lock()
	defer w.mux.Unlock()
	w.fsync = append(w.fsync, syncMsg{
		sync:  sync,
		count: w.pending,
	})
	w.pending = 0

	w.cond.Signal()
}

func (w *writer) Run() error {
	var (
		buf   [1024]writeMsg
		n     int
		err   error
		fsync *txWriteSync
		done  bool
	)

	traceln("start async writer")
	defer traceln("stop async writer")

	for {
		n, fsync, done = w.nextCommand(buf[:])
		if done {
			break
		}

		traceln("writer message: ", n, fsync != nil, done)

		// TODO: use vector IO if possible
		msgs := buf[:n]
		sort.Slice(msgs, func(i, j int) bool {
			return msgs[i].id < msgs[j].id
		})

		for _, msg := range msgs {
			if err != nil {
				traceln("done error")

				msg.sync.err = err
				msg.sync.wg.Done()
				continue
			}

			off := uint64(msg.id) * uint64(w.pageSize)
			tracef("write at(id=%v, off=%v, len=%v)\n", msg.id, off, len(msg.buf))

			err = writeAt(w.target, msg.buf, int64(off))
			if err != nil {
				msg.sync.err = err
			}

			traceln("done send")
			msg.sync.Release()
		}

		if fsync != nil {
			if err == nil {
				if err = w.target.Sync(); err != nil {
					fsync.err = err
				}
			}

			traceln("done fsync")
			fsync.Release()
		}

		if err != nil {
			break
		}
	}

	if done {
		return err
	}

	// file still active, but we're facing errors -> stop writing and propagate
	// last error to all transactions.
	for {
		n, fsync, done = w.nextCommand(buf[:])
		if done {
			break
		}

		traceln("ignoring writer message: ", n, fsync != nil, done)

		for _, msg := range buf[:n] {
			msg.sync.err = err
			msg.sync.Release()
		}
		if fsync != nil {
			fsync.err = err
			fsync.Release()
		}
	}

	return err
}

func (w *writer) nextCommand(buf []writeMsg) (int, *txWriteSync, bool) {
	w.mux.Lock()
	defer w.mux.Unlock()

	traceln("async writer: wait next command")
	defer traceln("async writer: received next command")

	for {
		if w.done {
			return 0, nil, true
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
		traceln("check fsync: ", len(w.fsync))

		if len(w.fsync) > 0 {
			msg := w.fsync[0]

			// number of outstanding scheduled writes before fsync
			outstanding := msg.count - w.published
			traceln("outstanding:", outstanding)

			if outstanding <= max { // -> fsync
				max, sync = outstanding, msg.sync

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

		return n, sync, false
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

func (s *txWriteSync) Wait() error {
	s.wg.Wait()
	return s.err
}

func writeAt(out io.WriterAt, buf []byte, off int64) error {
	for len(buf) > 0 {
		n, err := out.WriteAt(buf, off)
		if err != nil {
			return err
		}

		off += int64(n)
		buf = buf[n:]
	}
	return nil
}
