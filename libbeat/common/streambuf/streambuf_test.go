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

//go:build !integration

package streambuf

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)

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
	assert.NoError(t, err)
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)
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

// TestReuseAppendCollectCycle exercises the write/collect/reset cycle a line
// reader performs on a reuse-enabled buffer: append a "line", collect it, copy
// it out (a safe consumer), reset, repeat with varying sizes. It asserts the
// collected content is always correct and that the backing array is reused
// (capacity stabilizes) rather than reallocated on every line.
func TestReuseAppendCollectCycle(t *testing.T) {
	b := New(nil)
	b.SetReuse(true)

	// Line lengths chosen to grow, shrink, and occasionally exceed the current
	// capacity so every appendReuse branch (in-place, compact, grow) is hit.
	// The largest line (16384) appears before later smaller-or-equal lines so we
	// can prove those reuse the existing array rather than reallocating.
	lengths := []int{8, 200, 50, 16384, 16384, 40, 16384, 1, 9000, 9000}
	seed := byte(0)

	var baseAfterMax []byte // b.base captured right after the high-water line

	for i, n := range lengths {
		// Build the line and feed it in two writes to mimic chunked decoding.
		line := make([]byte, n)
		for j := range line {
			line[j] = seed
			seed++
		}
		half := n / 2
		require.NoError(t, b.Append(line[:half]))
		require.NoError(t, b.Append(line[half:]))
		require.NoError(t, checkInvariants(b))

		got, err := b.Collect(b.Len())
		require.NoError(t, err)
		// A safe consumer copies before reading on; the next iteration's append
		// is allowed to overwrite the collected backing array.
		require.Equal(t, line, append([]byte(nil), got...), "line %d content mismatch", i)

		b.Reset()
		require.NoError(t, checkInvariants(b))

		// Once the high-water (16384) array exists, every later line that fits
		// must keep the SAME backing array — i.e. no reallocation.
		if n == 16384 {
			require.GreaterOrEqual(t, cap(b.base), 16384)
			baseAfterMax = b.base
		} else if baseAfterMax != nil && n <= cap(baseAfterMax) {
			require.Equal(t, cap(baseAfterMax), cap(b.base),
				"line %d (%d bytes) reallocated instead of reusing the array", i, n)
			require.Same(t, &baseAfterMax[:1][0], &b.base[:1][0],
				"line %d (%d bytes) replaced the backing array", i, n)
		}
	}
}

// TestReuseDisabledUnchanged verifies that without SetReuse the buffer behaves
// exactly as before (no base tracking, content still correct).
func TestReuseDisabledUnchanged(t *testing.T) {
	b := New(nil)
	for _, s := range []string{"alpha\n", "beta\n", "gamma\n"} {
		require.NoError(t, b.Append([]byte(s)))
		got, err := b.Collect(b.Len())
		require.NoError(t, err)
		require.Equal(t, s, string(got))
		b.Reset()
		require.Nil(t, b.base, "reuse-disabled buffer must not track base")
	}
}
