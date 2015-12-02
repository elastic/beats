// The streambuf module provides helpers for buffering multiple packet payloads
// and some general parsing functions. All parsing functions are re-entrant,
// that is if a parse function fails due do not having buffered enough bytes yet
// (error value ErrNoMoreBytes) the parser can be called again after appending more
// bytes to the buffer. Parsers potentially reading large amount of bytes might
// remember the last position.
// Additionally a Buffer can be marked as fixed. Fixed buffers to not support
// adding new data, plus ErrNoMoreBytes will never be returned. Instead if a parser
// decides it need more bytes ErrUnexpectedEOB will be returned.
//
// Error handling:
// All functions that might fail, will return an error. The last error reported
// will be stored with the buffer itself. Instead of checking every single error
// one can use the Failed() and Err() methods to check if the buffer is still in a
// valid state and all parsing was successfull.
package streambuf

import (
	"github.com/elastic/libbeat/logp"

	"bytes"
	"errors"
)

// Error returned if Append or Write operation is not allowed due to the buffer
// being fixed
var ErrOperationNotAllowed = errors.New("Operation not allowed")

var ErrOutOfRange = errors.New("Data access out of range")

// Parse operation can not be continued. More bytes required. Only returned if
// buffer is not fixed
var ErrNoMoreBytes = errors.New("No more bytes")

// Parse operation failed cause of buffer snapped short + buffer is fixed.
var ErrUnexpectedEOB = errors.New("unexpected end of buffer")

var ErrExpectedByteSequenceMismatch = errors.New("expected byte sequence did not match")

// A Buffer is a variable sized buffer of bytes with Read, Write and simple
// parsing methods. The zero value is an empty buffer ready for use.
//
// A Buffer can be marked as fixed. In this case no data can be appended to the
// buffer anymore and parser/reader methods will fail with ErrUnexpectedEOB if they
// would expect more bytes to come. Mark buffers fixed if some slice was separated
// for further parsing first.
type Buffer struct {
	data  []byte
	err   error
	fixed bool

	// Internal parser state offsets.
	// Offset is the position a parse might continue to work at when called
	// again (e.g. usefull for parsing tcp streams.). The mark is used to remember
	// the position last parse operation ended at. The variable available is used
	// for faster lookup
	// Invariants:
	//    (1) 0 <= mark <= offset
	//    (2) 0 <= available <= len(data)
	//    (3) available = len(data) - mark
	mark, offset, available int
}

// Init initializes a zero buffer with some byte slice being retained by the
// buffer. Usage of Init is optional as zero value Buffer is already in valid state.
func (b *Buffer) Init(d []byte, fixed bool) {
	b.data = d
	b.available = len(d)
	b.fixed = fixed
}

// New creates new extensible buffer from data slice being retained by the buffer.
func New(data []byte) *Buffer {
	return &Buffer{
		data:      data,
		fixed:     false,
		available: len(data),
	}
}

// NewFixed create new fixed buffer from data slice being retained by the buffer.
func NewFixed(data []byte) *Buffer {
	return &Buffer{
		data:      data,
		fixed:     true,
		available: len(data),
	}
}

// Snapshot creates a snapshot of buffers current state. Use in conjunction with
// Restore to get simple backtracking support. Between Snapshot and Restore the
// Reset method MUST not be called, as restored buffer will be in invalid state
// after.
func (b *Buffer) Snapshot() *Buffer {
	tmp := *b
	return &tmp
}

// Restore restores a buffers state. Use in conjunction with
// Snapshot to get simple backtracking support. Between Snapshot and Restore the
// Reset method MUST not be called, as restored buffer will be in invalid state
// after.
func (b *Buffer) Restore(snapshot *Buffer) {
	b.err = snapshot.err
	b.fixed = snapshot.fixed
	b.mark = snapshot.mark
	b.offset = snapshot.offset
	b.available = snapshot.available
}

func (b *Buffer) doAppend(data []byte, retainable bool) error {
	if b.fixed {
		return b.SetError(ErrOperationNotAllowed)
	}
	if b.err != nil && b.err != ErrNoMoreBytes {
		return b.err
	}

	if len(b.data) == 0 {
		if retainable {
			b.data = data
		} else {
			b.data = make([]byte, len(data))
			copy(b.data, data)
		}
	} else {
		b.data = append(b.data, data...)
	}
	b.available += len(data)

	// reset error status (continue parsing)
	if b.err == ErrNoMoreBytes {
		b.err = nil
	}

	return nil
}

// Append will append the given data to the buffer. If Buffer is fixed
// ErrOperationNotAllowed will be returned.
func (b *Buffer) Append(data []byte) error {
	return b.doAppend(data, true)
}

// Fix marks a buffer as fixed. No more data can be added to the buffer and
// parse operation might fail if they expect more bytes.
func (b *Buffer) Fix() {
	b.fixed = true
}

// Total returns the total number of bytes stored in the buffer
func (b *Buffer) Total() int {
	return len(b.data)
}

// Avail checks if count bytes are available for reading from the buffer.
func (b *Buffer) Avail(count int) bool {
	return count <= b.available
}

// Len returns the number of bytes of the unread portion.
func (b *Buffer) Len() int {
	return b.available
}

// LeftBehind returns the number of bytes a re-entrant but not yet finished
// parser did already read.
func (b *Buffer) LeftBehind() int {
	return b.offset - b.mark
}

// BufferConsumed returns the number of bytes already consumed since last call to Reset.
func (b *Buffer) BufferConsumed() int {
	return b.mark
}

// Advance will advance the buffers read pointer by count bytes. Returns
// ErrNoMoreBytes or ErrUnexpectedEOB if count bytes are not available.
func (b *Buffer) Advance(count int) error {
	if !b.Avail(count) {
		return b.bufferEndError()
	}
	b.mark += count
	b.offset = b.mark
	b.available -= count
	return nil
}

// Failed returns true if buffer is in failed state. If buffer is in failed
// state, almost all buffer operations will fail
func (b *Buffer) Failed() bool {
	failed := b.err != nil
	if failed {
		logp.Debug("streambuf", "buf parser already failed with: %s", b.err)
	}
	return failed
}

// Returns the error value of the last failed operation.
func (b *Buffer) Err() error {
	return b.err
}

// Check if n bytes are addressable in the buffer offset at b.mark and
// increases either the length or allocates bigger slice if necessary
func (b *Buffer) ensureLen(n int) {
	delta := n - b.available
	if delta <= 0 {
		// no additional space required:
		return
	}

	// newly available bytes
	b.available += delta

	total := len(b.data) + delta
	if total <= cap(b.data) {
		// enough space in slice -> grow it
		b.data = b.data[0:total]
		return
	}

	tmp := make([]byte, total)
	copy(tmp, b.data)
	b.data = tmp
}

// return slice to write to starting at off + b.mark with given length.
func (b *Buffer) sliceAt(off, len int) []byte {
	off += b.mark
	end := off + len
	b.ensureLen(end - b.mark)
	return b.data[off:end]
}

// Consume removes the first n bytes (special variant of Reset) from the
// beginning of the buffer, if at least n bytes have already been processed.
// Returns the byte slice of all bytes being removed from the buffer.
// If total buffer is < n, ErrOutOfRange will be reported or ErrOutOfRange if
// not enough bytes have been processed yet.
func (b *Buffer) Consume(n int) ([]byte, error) {
	if n > len(b.data) {
		return nil, ErrOutOfRange
	}

	new_mark := b.mark - n
	if new_mark < 0 {
		return nil, ErrOutOfRange
	}

	old := b.data[:n]
	b.data = b.data[n:]
	b.mark = new_mark
	b.offset -= n
	b.available = len(b.data) - b.mark
	return old, nil
}

// Reset remove all bytes already processed from the buffer. Use Reset after
// processing message to limit amount of buffered data.
func (b *Buffer) Reset() {
	b.data = b.data[b.mark:]
	b.offset -= b.mark
	b.mark = 0
	b.available = len(b.data)
	b.err = nil
}

// BufferedBytes returns all buffered bytes since last reset.
func (b *Buffer) BufferedBytes() []byte {
	return b.data
}

// Bytes returns all bytes not yet processed. The read counters are not advanced
// yet. For example use with fixed Buffer for simple lookahead.
//
// Note:
// The read markers are not advanced. If rest of buffer should be
// processed, call Advance immediately.
func (b *Buffer) Bytes() []byte {
	return b.data[b.mark:]
}

func (b *Buffer) bufferEndError() error {
	if b.fixed {
		return b.SetError(ErrUnexpectedEOB)
	} else {
		return b.SetError(ErrNoMoreBytes)
	}
}

// SetError marks a buffer as failed. Append and parse operations will fail with
// err. SetError returns err directly.
func (b *Buffer) SetError(err error) error {
	b.err = err
	return err
}

// Collect tries to collect count bytes from the buffer and updates the read
// pointers. If the buffer is in failed state or count bytes are not available
// an error will be returned.
func (b *Buffer) Collect(count int) ([]byte, error) {
	if b.Failed() {
		return nil, b.err
	}

	if !b.Avail(count) {
		return nil, b.bufferEndError()
	}

	data := b.data[b.mark : b.mark+count]
	b.Advance(count)
	return data, nil
}

// CollectWithDelimiter collects count bytes and checks delim will immediately
// follow the byte sequence. Returns count bytes without delim.
// If delim is not matched ErrExpectedByteSequenceMismatch will be raised.
func (b *Buffer) CollectWithSuffix(count int, delim []byte) ([]byte, error) {
	total := count + len(delim)
	if b.Failed() {
		return nil, b.err
	}

	if !b.Avail(total) {
		return nil, b.bufferEndError()
	}

	end := b.mark + count
	if !bytes.HasPrefix(b.data[end:], delim) {
		return nil, b.SetError(ErrExpectedByteSequenceMismatch)
	}

	data := b.data[b.mark : b.mark+count]
	b.Advance(total)
	return data, nil
}

// Index returns offset of seq in unprocessed buffer.
// Returns -1 if seq can not be found.
func (b *Buffer) Index(seq []byte) int {
	return b.IndexFrom(0, seq)
}

// IndexFrom returns offset of seq in unprocessed buffer start at from.
// Returns -1 if seq can not be found.
func (b *Buffer) IndexFrom(from int, seq []byte) int {
	if b.err != nil {
		return -1
	}

	idx := bytes.Index(b.data[b.mark+from:], seq)
	if idx < 0 {
		return -1
	}

	return idx + from + b.mark
}

// IndexByte returns offset of byte in unprocessed buffer.
// Returns -1 if byte not in buffer.
func (b *Buffer) IndexByte(byte byte) int {
	if b.err != nil {
		return -1
	}

	idx := bytes.IndexByte(b.data[b.mark:], byte)
	if idx < 0 {
		return -1
	}
	return idx + b.mark
}

// CollectUntil collects all bytes until delim was found (including delim).
func (b *Buffer) CollectUntil(delim []byte) ([]byte, error) {
	if b.err != nil {
		return nil, b.err
	}

	idx := bytes.Index(b.data[b.mark:], delim)
	if idx < 0 {
		return nil, b.bufferEndError()
	}

	end := b.mark + idx + len(delim)
	data := b.data[b.mark:end]
	b.Advance(len(data))
	return data, nil
}

// CollectUntilByte collects all bytes until delim was found (including delim).
func (b *Buffer) CollectUntilByte(delim byte) ([]byte, error) {
	if b.err != nil {
		return nil, b.err
	}

	idx := bytes.IndexByte(b.data[b.offset:], delim)
	if idx < 0 {
		b.offset = b.mark + b.available
		return nil, b.bufferEndError()
	}

	end := b.offset + idx + 1
	data := b.data[b.mark:end]
	b.Advance(len(data))
	return data, nil
}
