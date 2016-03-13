// +build !integration

package streambuf

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var ErrMarkInvariant = errors.New("mark value not within limits")
var ErrAvailableInvariant = errors.New("available value not within limits")
var ErrSizesInvariant = errors.New("available and mark values not in sync")

var ErrTest = errors.New("test")

func checkInvariants(b *Buffer) error {
	if !(0 <= b.mark && b.mark <= b.offset) {
		return ErrMarkInvariant
	}
	if !(0 <= b.available && b.available <= len(b.data)) {
		return ErrAvailableInvariant
	}
	if !(b.available == len(b.data)-b.mark) {
		return ErrSizesInvariant
	}

	return nil
}

func (b *Buffer) checkInvariants(t *testing.T) {
	assert.Nil(t, checkInvariants(b))
}

func Test_InvariantsOnNew(t *testing.T) {
	var b1 Buffer
	b1.checkInvariants(t)

	var b2 Buffer
	b2.Init([]byte("test"), false)
	b2.checkInvariants(t)

	New([]byte("test")).checkInvariants(t)

	NewFixed([]byte("test")).checkInvariants(t)
}

func Test_ErrorHandling(t *testing.T) {
	b := New(nil)
	b.checkInvariants(t)

	assert.False(t, b.Failed())
	assert.Nil(t, b.Err())

	err := b.SetError(ErrTest)
	assert.Equal(t, ErrTest, err)
	assert.True(t, b.Failed())
	assert.Equal(t, ErrTest, b.Err())
}

func Test_SnapshotRestore(t *testing.T) {
	b := NewFixed([]byte("test test"))
	snapshot := b.Snapshot()

	err := b.Advance(5)
	assert.Equal(t, 5, b.BufferConsumed())
	assert.Equal(t, 4, b.Len())
	assert.Nil(t, err)
	assert.False(t, b.Failed())

	b.Restore(snapshot)
	b.checkInvariants(t)
	assert.Equal(t, 9, b.Len())
	assert.Equal(t, 0, b.BufferConsumed())
}

func Test_SnapshotRestoreAfterErr(t *testing.T) {
	b := NewFixed([]byte("test test"))
	snapshot := b.Snapshot()

	err := b.Advance(20)
	assert.True(t, b.Failed())
	assert.Error(t, err)
	assert.Error(t, b.Err())

	b.Restore(snapshot)
	b.checkInvariants(t)
	assert.False(t, b.Failed())
	assert.Nil(t, b.Err())
}

func Test_AppendNil(t *testing.T) {
	b := NewFixed(nil)
	b.Append(nil)
	b.checkInvariants(t)
	assert.Equal(t, 0, b.Len())
}

func Test_AppendRetainsBuffer(t *testing.T) {
	d := []byte("test")
	b := New(nil)

	b.Append(d)
	d[0] = 'a'
	x, _ := b.Collect(1)
	b.checkInvariants(t)
	assert.False(t, b.Failed())
	assert.Equal(t, d[0], x[0])
}

func Test_AppendOnFixed(t *testing.T) {
	b := NewFixed([]byte("abc"))

	err := b.Append([]byte("def"))
	assert.Equal(t, ErrOperationNotAllowed, err)
	assert.True(t, b.Failed())
	assert.Equal(t, err, b.Err())
}

func Test_AppendOnFixedLater(t *testing.T) {
	b := New([]byte("abc"))

	err := b.Append([]byte("def"))
	assert.Nil(t, err)

	b.Fix()
	err = b.Append([]byte("def"))
	b.checkInvariants(t)
	assert.Equal(t, ErrOperationNotAllowed, err)
	assert.True(t, b.Failed())
	assert.Equal(t, err, b.Err())
}

func Test_AppendOnFailed(t *testing.T) {
	b := New([]byte("abc"))
	b.SetError(ErrTest)
	err := b.Append([]byte("def"))
	assert.Equal(t, ErrTest, err)
}

func Test_AppendAfterNoMoreBytes(t *testing.T) {
	b := New([]byte("a"))

	err := b.Advance(5)
	assert.Equal(t, ErrNoMoreBytes, err)

	err = b.Append([]byte(" test"))
	assert.Nil(t, err)
	assert.False(t, b.Failed())
}

func Test_AvailAndLenConsiderRead(t *testing.T) {
	b := New([]byte("test"))
	b.Advance(3)
	b.checkInvariants(t)
	assert.Equal(t, 4, b.Total())
	assert.Equal(t, 1, b.Len())
	assert.Equal(t, 3, b.BufferConsumed())
}

func Test_AvailAndLenConsiderReset(t *testing.T) {
	b := New([]byte("test"))
	b.Advance(3)
	b.Reset()
	b.checkInvariants(t)
	assert.Equal(t, 1, b.Total())
	assert.Equal(t, 1, b.Len())
	assert.Equal(t, 0, b.BufferConsumed())
}

func Test_ConsumeData(t *testing.T) {
	b := New([]byte("test"))
	b.Advance(3)
	b.Consume(2)
	b.checkInvariants(t)
	assert.Equal(t, 2, b.Total())
	assert.Equal(t, 1, b.Len())
	assert.Equal(t, 1, b.BufferConsumed())
}

func Test_ConsumeFailed(t *testing.T) {
	b := New([]byte("test"))
	snapshot := b.Snapshot()

	_, err := b.Consume(100)
	assert.Equal(t, ErrOutOfRange, err)

	b.Restore(snapshot)
	assert.False(t, b.Failed())

	_, err = b.Consume(3)
	assert.Equal(t, ErrOutOfRange, err)
}

func Test_ByteGetUnconsumed(t *testing.T) {
	b := New([]byte("test"))
	b.Advance(3)
	d := b.Bytes()

	b.checkInvariants(t)
	assert.Equal(t, 3, b.mark)
	assert.Equal(t, 1, len(d))
	assert.True(t, 't' == d[0])
}

func Test_ResetEmpty(t *testing.T) {
	b := New(nil)
	b.Reset()
	b.checkInvariants(t)
}

func Test_ResetWhileParsing(t *testing.T) {
	b := New([]byte("test"))
	b.Advance(1)
	b.offset += 2
	b.checkInvariants(t)
	assert.Equal(t, 3, b.offset)

	b.Reset()
	b.checkInvariants(t)
	assert.Equal(t, 2, b.offset)
}

func Test_CollectData(t *testing.T) {
	b := New([]byte("test"))
	d, err := b.Collect(2)

	b.checkInvariants(t)
	assert.Nil(t, err)
	assert.Equal(t, []byte("te"), d)
}

func Test_CollectFailed(t *testing.T) {
	b := New([]byte("test"))
	b.SetError(ErrTest)

	d, err := b.Collect(2)
	assert.Equal(t, ErrTest, err)
	assert.Nil(t, d)
}

func Test_CollectNoData(t *testing.T) {
	b := New(nil)

	d, err := b.Collect(2)
	assert.True(t, b.Failed())
	assert.Equal(t, ErrNoMoreBytes, err)
	assert.Nil(t, d)
}

func Test_CollectFixedNoData(t *testing.T) {
	b := NewFixed(nil)

	d, err := b.Collect(2)
	assert.True(t, b.Failed())
	assert.Equal(t, ErrUnexpectedEOB, err)
	assert.Nil(t, d)
}

func Test_CollectWithSuffixData(t *testing.T) {
	b := New([]byte("test\r\ntest"))

	d, err := b.CollectWithSuffix(4, []byte("\r\n"))
	b.checkInvariants(t)
	assert.False(t, b.Failed())
	assert.Nil(t, err)
	assert.Equal(t, []byte("test"), d)
}

func Test_CollectWithSuffixFail(t *testing.T) {
	b := New([]byte("test\n\ntest"))

	d, err := b.CollectWithSuffix(4, []byte("\r\n"))
	assert.True(t, b.Failed())
	assert.Nil(t, d)
	assert.Equal(t, ErrExpectedByteSequenceMismatch, err)
}

func Test_CollectWithSuffixFailed(t *testing.T) {
	b := New([]byte("test\r\ntest"))
	b.SetError(ErrTest)

	d, err := b.CollectWithSuffix(4, []byte("\r\n"))
	assert.Equal(t, ErrTest, err)
	assert.Nil(t, d)
}

func Test_CollectWithSuffixNoData(t *testing.T) {
	b := New(nil)

	d, err := b.CollectWithSuffix(4, []byte("\r\n"))
	assert.True(t, b.Failed())
	assert.Equal(t, ErrNoMoreBytes, err)
	assert.Nil(t, d)
}

func Test_CollectWithSuffixFixedNoData(t *testing.T) {
	b := NewFixed(nil)

	d, err := b.CollectWithSuffix(4, []byte("\r\n"))
	assert.True(t, b.Failed())
	assert.Equal(t, ErrUnexpectedEOB, err)
	assert.Nil(t, d)
}
