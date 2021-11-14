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
// +build !integration

package streambuf

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ImplementsInterfaces(t *testing.T) {
	b := New(nil)
	var _ io.Reader = b
	var _ io.Writer = b
	var _ io.ReadWriter = b
	var _ io.ReaderFrom = b
	var _ io.ByteReader = b
	var _ io.ByteScanner = b
	var _ io.ByteWriter = b
	var _ io.RuneReader = b
}

func Test_ReadByteEOFCheck(t *testing.T) {
	b := New(nil)
	n, err := b.ReadByte()
	assert.Equal(t, byte(0), n)
	assert.Equal(t, io.EOF, err)

	b = New(nil)
	b.SetError(ErrNoMoreBytes)
	n, err = b.ReadByte()
	assert.Equal(t, byte(0), n)
	assert.Equal(t, io.EOF, err)

	b = New(nil)
	b.SetError(ErrUnexpectedEOB)
	n, err = b.ReadByte()
	assert.Equal(t, byte(0), n)
	assert.Equal(t, io.EOF, err)

	b = New(nil)
	b.SetError(ErrTest)
	n, err = b.ReadByte()
	assert.Equal(t, byte(0), n)
	assert.Equal(t, ErrTest, err)
}

func Test_ReadByteOK(t *testing.T) {
	b := New([]byte{1})
	v, err := b.ReadByte()
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.Equal(t, byte(1), v)

	_, err = b.ReadByte()
	assert.Equal(t, io.EOF, err)
}

func Test_ReadUnreadByteOK(t *testing.T) {
	b := New([]byte{1, 2})
	v, err := b.ReadByte()
	b.checkInvariants(t)
	assert.Equal(t, byte(1), v)
	assert.NoError(t, err)

	err = b.UnreadByte()
	assert.NoError(t, err)
	assert.Equal(t, 2, b.Len())
}

func Test_ReadUnreadByteErrCheck(t *testing.T) {
	b := New(nil)
	b.SetError(ErrTest)
	err := b.UnreadByte()
	b.checkInvariants(t)
	assert.Equal(t, ErrTest, err)
}

func Test_UnreadByteFail(t *testing.T) {
	b := New(nil)
	err := b.UnreadByte()
	b.checkInvariants(t)
	assert.Equal(t, ErrOutOfRange, err)
}

func Test_UnreadAfterEOFOK(t *testing.T) {
	b := New([]byte{1})

	b.ReadByte()
	_, err := b.ReadByte()
	assert.Equal(t, io.EOF, err)

	err = b.UnreadByte()
	assert.NoError(t, err)
}

func Test_WriteByte(t *testing.T) {
	b := New(nil)

	err := b.WriteByte(1)
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.Equal(t, 1, b.Len())
	assert.Equal(t, byte(1), b.Bytes()[0])
}

func Test_WriteByteEOFCheck(t *testing.T) {
	b := New(nil)

	_, err := b.ReadByte()
	assert.Equal(t, io.EOF, err)

	err = b.WriteByte(1)
	b.checkInvariants(t)
	assert.NoError(t, err)
}

func Test_WriteByteFixedFail(t *testing.T) {
	b := NewFixed(nil)
	err := b.WriteByte(1)
	b.checkInvariants(t)
	assert.Equal(t, ErrOperationNotAllowed, err)
}

func Test_ReadBufSmaller(t *testing.T) {
	b := New([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	tmp := make([]byte, 5)

	n, err := b.Read(tmp)
	b.checkInvariants(t)
	assert.Equal(t, 5, n)
	assert.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3, 4, 5}, tmp[:n])

	n, err = b.Read(tmp)
	b.checkInvariants(t)
	assert.Equal(t, 3, n)
	assert.NoError(t, err)
	assert.Equal(t, []byte{6, 7, 8}, tmp[:n])

	n, err = b.Read(tmp)
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)
}

func Test_ReadBufBigger(t *testing.T) {
	b := New([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	tmp := make([]byte, 10)

	n, err := b.Read(tmp)
	b.checkInvariants(t)
	assert.Equal(t, 8, n)
	assert.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3, 4, 5, 6, 7, 8}, tmp[:n])

	n, err = b.Read(tmp)
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)
}

func Test_ReadOnFailed(t *testing.T) {
	b := New([]byte{1, 2, 3})
	b.SetError(ErrTest)
	tmp := make([]byte, 10)
	_, err := b.Read(tmp)
	assert.Equal(t, ErrTest, err)
}

func Test_WriteOK(t *testing.T) {
	b := New(nil)
	n, err := b.Write([]byte{1, 2, 3})
	b.checkInvariants(t)
	assert.Equal(t, 3, n)
	assert.NoError(t, err)
	assert.Equal(t, 3, b.Len())
}

func Test_WriteDoesNotRetain(t *testing.T) {
	tmp := []byte{1, 2, 3}

	b := New(nil)
	n, err := b.Write(tmp)
	b.checkInvariants(t)
	assert.Equal(t, 3, n)
	assert.NoError(t, err)

	b.Bytes()[0] = 'a'
	assert.Equal(t, byte(1), tmp[0])
}

func Test_WriteFail(t *testing.T) {
	b := New(nil)
	b.SetError(ErrTest)
	_, err := b.Write([]byte{1})
	assert.Equal(t, ErrTest, err)
}

func Test_WriteNil(t *testing.T) {
	b := New([]byte{1, 2, 3})
	n, err := b.Write(nil)
	assert.Equal(t, 0, n)
	assert.NoError(t, err)
}

func Test_ReadFromOK(t *testing.T) {
	b := New(nil)
	from := New([]byte{1, 2, 3, 4})

	n, err := b.ReadFrom(from)
	assert.Equal(t, int64(4), n)
	assert.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3, 4}, b.Bytes())

	// check buffers are not retained
	b.Bytes()[0] = 'a'
	assert.Equal(t, byte(1), from.BufferedBytes()[0])

	// check from is really eof
	_, err = from.ReadByte()
	assert.Equal(t, io.EOF, err)
}

func Test_ReadFromIfEOF(t *testing.T) {
	b := New(nil)
	from := New([]byte{1, 2, 3, 4})

	// move buffer into EOF state
	_, err := b.ReadByte()
	assert.Equal(t, io.EOF, err)

	// copy from
	n, err := b.ReadFrom(from)
	assert.Equal(t, int64(4), n)
	assert.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3, 4}, b.Bytes())

	// check buffers are not retained
	b.Bytes()[0] = 'a'
	assert.Equal(t, byte(1), from.BufferedBytes()[0])

	// check from is really eof
	_, err = from.ReadByte()
	assert.Equal(t, io.EOF, err)
}

func Test_ReadFromFailOnFixed(t *testing.T) {
	b := NewFixed(nil)
	from := NewFixed([]byte{1, 2, 3, 4})

	n, err := b.ReadFrom(from)
	assert.Equal(t, int64(0), n)
	assert.Equal(t, err, ErrOperationNotAllowed)
}

func Test_ReadRuneOK(t *testing.T) {
	b := New([]byte("xäüö"))
	r, s, err := b.ReadRune()
	assert.NoError(t, err)
	assert.Equal(t, 'x', r)
	assert.Equal(t, 1, s)

	r, s, err = b.ReadRune()
	assert.NoError(t, err)
	assert.Equal(t, 'ä', r)
	assert.Equal(t, 2, s)
}

func Test_ReadRuneEOFCheck(t *testing.T) {
	b := New(nil)
	_, _, err := b.ReadRune()
	assert.Equal(t, io.EOF, err)
}

func Test_ReadRuneFailed(t *testing.T) {
	b := New(nil)
	b.SetError(ErrTest)
	_, _, err := b.ReadRune()
	assert.Equal(t, ErrTest, err)
}

func Test_ReadAtOK(t *testing.T) {
	b := New([]byte{1, 2, 3, 4})
	b.Advance(1)

	tmp := make([]byte, 2)
	n, err := b.ReadAt(tmp, 1)
	assert.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, []byte{3, 4}, tmp[:n])

	n, err = b.ReadAt(tmp, 2)
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []byte{4}, tmp[:n])
}

func Test_ReadAtOutOfRange(t *testing.T) {
	b := New([]byte{1, 2, 3, 4})
	b.Advance(1)

	tmp := make([]byte, 2)
	_, err := b.ReadAt(tmp, -1)
	assert.Equal(t, ErrOutOfRange, err)

	_, err = b.ReadAt(tmp, 10)
	assert.Equal(t, ErrOutOfRange, err)
}

func Test_WriteAtToNil(t *testing.T) {
	b := New(nil)
	n, err := b.WriteAt([]byte{1, 2, 3}, 4)
	assert.Equal(t, 3, n)
	assert.NoError(t, err)
}

func Test_WriteAtOverwrites(t *testing.T) {
	b := New([]byte{'a', 'b', 'c', 'd', 'e'})
	b.Advance(1)
	n, err := b.WriteAt([]byte{1, 2, 3}, 1)
	assert.Equal(t, 3, n)
	assert.NoError(t, err)
	assert.Equal(t, []byte{'b', 1, 2, 3}, b.Bytes())

	b = New(make([]byte, 3, 20))
	b.Advance(2)
	n, err = b.WriteAt([]byte{1, 2, 3}, 1)
	assert.Equal(t, 3, n)
	assert.NoError(t, err)
	assert.Equal(t, 4, b.Len())
	// assert.Equal(t, []byte{0, 1, 2, 3}, b.Bytes())
}
