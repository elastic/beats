package sys

// ByteBuffer is an expandable buffer backed by a byte slice.
type ByteBuffer struct {
	buf    []byte
	offset int
}

// NewByteBuffer creates a new ByteBuffer with an initial capacity of
// initialSize.
func NewByteBuffer(initialSize int) *ByteBuffer {
	return &ByteBuffer{buf: make([]byte, initialSize)}
}

// Write appends the contents of p to the buffer, growing the buffer as needed.
// The return value is the length of p; err is always nil.
func (b *ByteBuffer) Write(p []byte) (int, error) {
	if len(b.buf) < b.offset+len(p) {
		// Create a buffer larger than needed so we don't spend lots of time
		// allocating and copying.
		spaceNeeded := len(b.buf) - b.offset + len(p)
		largerBuf := make([]byte, 2*len(b.buf)+spaceNeeded)
		copy(largerBuf, b.buf[:b.offset])
		b.buf = largerBuf
	}
	n := copy(b.buf[b.offset:], p)
	b.offset += n
	return n, nil
}

// Reset resets the buffer to be empty. It retains the same underlying storage.
func (b *ByteBuffer) Reset() {
	b.offset = 0
	b.buf = b.buf[:cap(b.buf)]
}

// Bytes returns a slice of length b.Len() holding the bytes that have been
// written to the buffer.
func (b *ByteBuffer) Bytes() []byte {
	return b.buf[:b.offset]
}

// Len returns the number of bytes that have been written to the buffer.
func (b *ByteBuffer) Len() int {
	return b.offset
}
