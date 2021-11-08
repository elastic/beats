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
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_UntilCRLFOK(t *testing.T) {
	b := New([]byte("  test\r\n"))
	b.Advance(2)
	d, err := b.UntilCRLF()
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.False(t, b.Failed())
	assert.Equal(t, d, []byte("test"))
	assert.Equal(t, 0, b.Len())
}

func Test_UntilCRLFFailed(t *testing.T) {
	b := New([]byte("  test\r\nabc"))
	b.SetError(ErrTest)
	_, err := b.UntilCRLF()
	assert.Equal(t, ErrTest, err)
}

func Test_UntilCRLFCont(t *testing.T) {
	b := New([]byte("  test"))
	b.Advance(2)

	_, err := b.UntilCRLF()
	assert.Equal(t, ErrNoMoreBytes, err)

	err = b.Append([]byte("\r\nabc"))
	assert.NoError(t, err)
	assert.False(t, b.Failed())
	assert.Equal(t, 4, b.LeftBehind())

	d, err := b.UntilCRLF()
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.False(t, b.Failed())
	assert.Equal(t, d, []byte("test"))
	assert.Equal(t, 3, b.Len())
}

func Test_UntilCRLFOnlyCRThenCRLF(t *testing.T) {
	b := New([]byte("test\rtest\r\nabc"))
	d, err := b.UntilCRLF()
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.False(t, b.Failed())
	assert.Equal(t, d, []byte("test\rtest"))
	assert.Equal(t, 3, b.Len())
}

func Test_UntilCRLFOnlyCRThenCRLFWithCont(t *testing.T) {
	b := New([]byte("test\rtest\r"))

	_, err := b.UntilCRLF()
	assert.Equal(t, ErrNoMoreBytes, err)

	err = b.Append([]byte("\nabc"))
	assert.NoError(t, err)
	assert.False(t, b.Failed())
	assert.Equal(t, 9, b.LeftBehind())

	d, err := b.UntilCRLF()
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.False(t, b.Failed())
	assert.Equal(t, d, []byte("test\rtest"))
	assert.Equal(t, 3, b.Len())
}

func Test_IgnoreSymbolOK(t *testing.T) {
	b := New([]byte("  test"))
	err := b.IgnoreSymbol(' ')
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.False(t, b.Failed())
	assert.Equal(t, 4, b.Len())
}

func Test_IgnoreSymbolFailed(t *testing.T) {
	b := New([]byte("  test"))
	b.SetError(ErrTest)
	err := b.IgnoreSymbol(' ')
	assert.Equal(t, ErrTest, err)
}

func Test_IgnoreSymbolCont(t *testing.T) {
	b := New([]byte("    "))

	err := b.IgnoreSymbol(' ')
	assert.Equal(t, ErrNoMoreBytes, err)
	assert.Equal(t, 4, b.LeftBehind())

	b.Append([]byte("  test"))
	err = b.IgnoreSymbol(' ')
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.False(t, b.Failed())
	assert.Equal(t, 4, b.Len())
}

func Test_UntilSymbolOK(t *testing.T) {
	b := New([]byte("test "))
	d, err := b.UntilSymbol(' ', true)
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), d)
}

func Test_UntilSymbolFailed(t *testing.T) {
	b := New([]byte("test "))
	b.SetError(ErrTest)
	_, err := b.UntilSymbol(' ', true)
	assert.Equal(t, ErrTest, err)
}

func Test_UntilSymbolCont(t *testing.T) {
	b := New([]byte("tes"))

	_, err := b.UntilSymbol(' ', true)
	assert.Equal(t, ErrNoMoreBytes, err)
	assert.Equal(t, 3, b.LeftBehind())

	b.Append([]byte("t "))
	d, err := b.UntilSymbol(' ', true)
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), d)
}

func Test_UntilSymbolOrEnd(t *testing.T) {
	b := New([]byte("test"))
	d, err := b.UntilSymbol(' ', false)
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), d)
}

func Test_AsciiUintOK(t *testing.T) {
	b := New([]byte("123 "))
	v, err := b.UintASCII(false)
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.Equal(t, uint64(123), v)
}

func Test_AsciiUintFailed(t *testing.T) {
	b := New([]byte("123 "))
	b.SetError(ErrTest)
	_, err := b.UintASCII(false)
	assert.Equal(t, ErrTest, err)
}

func Test_AsciiUintNotDigit(t *testing.T) {
	b := New([]byte("test"))
	_, err := b.UintASCII(false)
	assert.Equal(t, ErrExpectedDigit, err)
}

func Test_AsciiUintEmpty(t *testing.T) {
	b := New([]byte(""))
	_, err := b.UintASCII(false)
	assert.Equal(t, ErrNoMoreBytes, err)
}

func Test_AsciiUintCont(t *testing.T) {
	b := New([]byte("12"))
	_, err := b.UintASCII(true)
	assert.Equal(t, ErrNoMoreBytes, err)

	b.Append([]byte("34 "))
	v, err := b.UintASCII(true)
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1234), v)
}

func Test_AsciiUintOrEndOK(t *testing.T) {
	b := New([]byte("12"))
	v, err := b.UintASCII(false)
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.Equal(t, uint64(12), v)
}

func Test_AsciiIntOK(t *testing.T) {
	b := New([]byte("123 "))
	v, err := b.IntASCII(false)
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.Equal(t, int64(123), v)
}

func Test_AsciiIntPosOK(t *testing.T) {
	b := New([]byte("+123 "))
	v, err := b.IntASCII(false)
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.Equal(t, int64(123), v)
}

func Test_AsciiIntNegOK(t *testing.T) {
	b := New([]byte("-123 "))
	v, err := b.IntASCII(false)
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.Equal(t, int64(-123), v)
}

func Test_AsciiIntFailed(t *testing.T) {
	b := New([]byte("123 "))
	b.SetError(ErrTest)
	_, err := b.IntASCII(false)
	assert.Equal(t, ErrTest, err)
}

func Test_AsciiIntNotDigit(t *testing.T) {
	b := New([]byte("test"))
	_, err := b.IntASCII(false)
	assert.Equal(t, ErrExpectedDigit, err)
}

func Test_AsciiIntEmpty(t *testing.T) {
	b := New([]byte(""))
	_, err := b.IntASCII(false)
	assert.Equal(t, ErrNoMoreBytes, err)
}

func Test_AsciiIntCont(t *testing.T) {
	b := New([]byte("12"))
	_, err := b.IntASCII(true)
	assert.Equal(t, ErrNoMoreBytes, err)

	b.Append([]byte("34 "))
	v, err := b.IntASCII(true)
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.Equal(t, int64(1234), v)
}

func Test_AsciiIntOrEndOK(t *testing.T) {
	b := New([]byte("12"))
	v, err := b.IntASCII(false)
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.Equal(t, int64(12), v)
}

func Test_AsciiMatchOK(t *testing.T) {
	b := New([]byte("match test"))
	r, err := b.MatchASCII([]byte("match"))
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.True(t, r)
	assert.Equal(t, 10, b.Len()) // check no bytes consumed
}

func Test_AsciiMatchNo(t *testing.T) {
	b := New([]byte("match test"))
	r, err := b.MatchASCII([]byte("batch"))
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.False(t, r)
	assert.Equal(t, 10, b.Len()) // check no bytes consumed
}

func Test_AsciiMatchCont(t *testing.T) {
	b := New([]byte("mat"))

	_, err := b.MatchASCII([]byte("match"))
	assert.Equal(t, ErrNoMoreBytes, err)

	b.Append([]byte("ch test"))
	r, err := b.MatchASCII([]byte("match"))
	b.checkInvariants(t)
	assert.NoError(t, err)
	assert.True(t, r)
	assert.Equal(t, 10, b.Len()) // check no bytes consumed
}

func Test_AsciiMatchFailed(t *testing.T) {
	b := New([]byte("match test"))
	b.SetError(ErrTest)
	_, err := b.MatchASCII([]byte("match"))
	assert.Equal(t, ErrTest, err)
}
