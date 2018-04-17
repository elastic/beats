package pq

import (
	"github.com/elastic/go-txfile"
)

// Queue implements the on-disk queue data structure. The queue requires a
// Delegate, so to start transactions at any time. The Queue provides a reader
// and writer. While it is safe to use the Reader and Writer concurrently, the
// Reader and Writer themselves are not thread-safe.
type Queue struct {
	accessor access

	// TODO: add support for multiple named readers with separate ACK handling.

	reader *Reader
	writer *Writer
	acker  *acker
}

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
	if delegate == nil {
		return nil, errNODelegate
	}

	accessor, err := makeAccess(delegate)
	if err != nil {
		return nil, err
	}

	q := &Queue{accessor: accessor}

	pageSize := delegate.PageSize()
	pagePool := newPagePool(pageSize)

	rootBuf, err := q.accessor.ReadRoot()
	if err != nil {
		return nil, err
	}

	root := castQueueRootPage(rootBuf[:])
	if root.version.Get() != queueVersion {
		return nil, errInvalidVersion
	}

	tracef("open queue: %p (pageSize: %v)\n", q, pageSize)
	traceQueueHeader(root)

	tail := q.accessor.ParsePosition(&root.tail)
	writer, err := newWriter(&q.accessor, pagePool,
		settings.WriteBuffer, tail, settings.Flushed)
	if err != nil {
		return nil, err
	}

	reader, err := newReader(&q.accessor)
	if err != nil {
		return nil, err
	}

	acker, err := newAcker(&q.accessor, settings.ACKed)
	if err != nil {
		return nil, err
	}

	q.reader = reader
	q.writer = writer
	q.acker = acker
	return q, nil
}

// Close will try to flush the current write buffer,
// but after closing the queue, no more reads or writes can be executed
func (q *Queue) Close() error {
	tracef("close queue %p\n", q)
	defer tracef("queue %p closed\n", q)

	q.reader.close()
	q.acker.close()
	return q.writer.close()
}

// Pending returns the total number of enqueued, but unacked events.
func (q *Queue) Pending() int {
	tx := q.accessor.BeginRead()
	defer tx.Close()

	hdr, err := q.accessor.RootHdr(tx)
	if err != nil {
		return -1
	}

	head := q.accessor.ParsePosition(&hdr.read)
	if head.page == 0 {
		head = q.accessor.ParsePosition(&hdr.head)
	}
	tail := q.accessor.ParsePosition(&hdr.tail)

	return int(tail.id - head.id)
}

// Writer returns the queue writer for inserting new events into the queue.
// A queue has only one single writer instance, which is returned by GetWriter.
// The writer is is not thread safe.
func (q *Queue) Writer() *Writer {
	return q.writer
}

// Reader returns the queue reader for reading a new events from the queue.
// A queue has only one single reader instance.
// The reader is not thread safe.
func (q *Queue) Reader() *Reader {
	return q.reader
}

// ACK signals the queue, the most n events at the front of the queue have been
// processed.
// The queue will try to remove these asynchronously.
func (q *Queue) ACK(n uint) error {
	return q.acker.handle(n)
}

// Active returns the number of active, not yet ACKed events.
func (q *Queue) Active() (uint, error) {
	return q.acker.Active()
}
