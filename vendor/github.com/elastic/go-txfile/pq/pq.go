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

package pq

import (
	"fmt"
	"unsafe"

	"github.com/elastic/go-txfile"
)

// Queue implements the on-disk queue data structure. The queue requires a
// Delegate, so to start transactions at any time. The Queue provides a reader
// and writer. While it is safe to use the Reader and Writer concurrently, the
// Reader and Writer themselves are not thread-safe.
type Queue struct {
	accessor access

	id        queueID
	version   uint32
	hdrOffset uintptr

	// TODO: add support for multiple named readers with separate ACK handling.

	pagePool *pagePool

	reader *Reader
	writer *Writer
	acker  *acker

	settings Settings
}

type queueID int

type position struct {
	page txfile.PageID
	off  int
	id   uint64
}

// Settings configures a queue when being instantiated with `New`.
type Settings struct {
	// Queue write buffer size. If a single event is bigger then the
	// write-buffer, the write-buffer will grow. In this case will the write
	// buffer be flushed and reset to its original size.
	WriteBuffer uint

	// Optional Flushed callback. Will be used to notify n events being
	// successfully committed.
	Flushed func(n uint)

	// Optional ACK callback. Will be use to notify number of events being successfully
	// ACKed and pages being freed.
	ACKed func(event, pages uint)

	Observer Observer
}

// MakeRoot prepares the queue header (empty queue).
// When creating a queue with `New`, the queue header must be available.
// Still, a delegate is allowed to create the queue header lazily.
func MakeRoot() [SzRoot]byte {
	var buf [SzRoot]byte
	qu := castQueueRootPage(buf[:])
	qu.version.Set(queueVersion)
	return buf
}

// New creates a new Queue. The delegate is required to access the file and
// start transactions. An error is returned if the delegate is nil, the queue
// header is invalid, some settings are invalid, or if some IO error occurred.
func New(delegate Delegate, settings Settings) (*Queue, error) {
	const op = "pq/new"

	if delegate == nil {
		return nil, errOp(op).of(InvalidParam).report("delegate must not be nil")
	}

	accessor, errKind := makeAccess(delegate)
	if errKind != NoError {
		return nil, errOp(op).of(errKind)
	}

	pageSize := delegate.PageSize()

	q := &Queue{
		accessor: accessor,
		settings: settings,
		pagePool: newPagePool(pageSize),
	}

	// use pointer address as ID for correlating error messages
	q.id = queueID(uintptr(unsafe.Pointer(q)))
	accessor.quID = q.id

	rootBuf, err := q.accessor.ReadRoot()
	if err != nil {
		return nil, wrapErr(op, err).of(InitFailed).
			report("failed to read queue header")
	}

	root := castQueueRootPage(rootBuf[:])
	if root.version.Get() != queueVersion {
		cause := &Error{
			kind: InitFailed,
			msg:  fmt.Sprintf("queue version %v", root.version.Get()),
		}
		return nil, wrapErr(op, cause).of(InitFailed)
	}

	tracef("open queue: %p (pageSize: %v)\n", q, pageSize)
	traceQueueHeader(root)

	q.version = root.version.Get()
	q.hdrOffset = q.accessor.RootFileOffset()
	q.onInit()
	return q, nil
}

func (q *Queue) onInit() {
	o := q.settings.Observer
	if o == nil {
		return
	}

	avail, _ := q.Active()
	o.OnQueueInit(q.hdrOffset, q.version, avail)
}

// Close will try to flush the current write buffer,
// but after closing the queue, no more reads or writes can be executed
func (q *Queue) Close() error {
	tracef("close queue %p\n", q)
	defer tracef("queue %p closed\n", q)

	if q.reader != nil {
		q.reader.close()
		q.reader = nil
	}

	if q.acker != nil {
		q.acker.close()
		q.acker = nil
	}

	var err error
	if q.writer != nil {
		err = q.writer.close()
		q.writer = nil
	}

	return err
}

// Pending returns the total number of enqueued, but unacked events.
func (q *Queue) Pending() (int, error) {
	tx, err := q.accessor.BeginRead()
	if err != nil {
		return -1, err
	}

	defer tx.Close()

	hdr, err := q.accessor.RootHdr(tx)
	if err != nil {
		return -1, err
	}

	head := q.accessor.ParsePosition(&hdr.read)
	if head.page == 0 {
		head = q.accessor.ParsePosition(&hdr.head)
	}
	tail := q.accessor.ParsePosition(&hdr.tail)

	return int(tail.id - head.id), nil
}

// Writer returns the queue writer for inserting new events into the queue.
// A queue has only one single writer instance, which is returned by GetWriter.
// The writer is is not thread safe.
func (q *Queue) Writer() (*Writer, error) {
	const op = "pq/get-writer"

	if q.writer != nil {
		return q.writer, nil
	}

	rootBuf, err := q.accessor.ReadRoot()
	if err != nil {
		return nil, q.accessor.errWrap(op, err)
	}

	root := castQueueRootPage(rootBuf[:])
	tail := q.accessor.ParsePosition(&root.tail)

	writeBuffer := q.settings.WriteBuffer
	flushed := q.settings.Flushed
	writer, err := newWriter(&q.accessor, q.hdrOffset, q.settings.Observer, q.pagePool, writeBuffer, tail, flushed)
	if err != nil {
		return nil, q.accessor.errWrap(op, err)
	}

	q.writer = writer
	return q.writer, nil
}

// Reader returns the queue reader for reading a new events from the queue.
// A queue has only one single reader instance.
// The reader is not thread safe.
func (q *Queue) Reader() *Reader {
	if q.reader == nil {
		q.reader = newReader(q.settings.Observer, &q.accessor)
	}
	return q.reader
}

// ACK signals the queue, the most n events at the front of the queue have been
// processed.
// The queue will try to remove these asynchronously.
func (q *Queue) ACK(n uint) error {
	return q.getAcker().handle(n)
}

// Active returns the number of active, not yet ACKed events.
func (q *Queue) Active() (uint, error) {
	return q.getAcker().Active()
}

func (q *Queue) getAcker() *acker {
	if q.acker == nil {
		q.acker = newAcker(&q.accessor, q.hdrOffset, q.settings.Observer, q.settings.ACKed)
	}
	return q.acker
}
