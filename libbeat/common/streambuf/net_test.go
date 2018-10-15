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

// +build !integration

package streambuf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ReadNetUint8NoData(t *testing.T) {
	b := New(nil)
	v, err := b.ReadNetUint8()
	assert.Equal(t, ErrNoMoreBytes, err)
	assert.Equal(t, uint8(0), v)
}

func Test_ReadNetUint8Failed(t *testing.T) {
	b := New(nil)
	b.SetError(ErrTest)
	v, err := b.ReadNetUint8()
	assert.Equal(t, ErrTest, err)
	assert.Equal(t, uint8(0), v)
}

func Test_ReadNetUint8Data(t *testing.T) {
	b := New([]byte{10})
	v, err := b.ReadNetUint8()
	b.checkInvariants(t)
	assert.Nil(t, err)
	assert.Equal(t, uint8(10), v)
}

func Test_ReadNetUint8AtFailed(t *testing.T) {
	b := New(nil)
	b.SetError(ErrTest)
	v, err := b.ReadNetUint8At(4)
	assert.Equal(t, ErrTest, err)
	assert.Equal(t, uint8(0), v)
}

func Test_ReadNetUint8AtInRange(t *testing.T) {
	b := New([]byte{1, 2, 3})
	v, err := b.ReadNetUint8At(2)
	assert.Nil(t, err)
	assert.Equal(t, uint8(3), v)
}

func Test_ReadNetUint8AtOutOfRange(t *testing.T) {
	b := New([]byte{1, 2, 3})
	v, err := b.ReadNetUint8At(3)
	assert.Equal(t, ErrNoMoreBytes, err)
	assert.Equal(t, uint8(0), v)
}

func Test_WriteNetUint8At(t *testing.T) {
	b := New(nil)
	err := b.WriteNetUint8At(10, 1)
	assert.Nil(t, err)

	b.Advance(1)
	tmp, err := b.ReadNetUint8()
	assert.Nil(t, err)
	assert.Equal(t, uint8(10), tmp)
}

func Test_ReadNetUint16NoData(t *testing.T) {
	b := New(nil)
	v, err := b.ReadNetUint16()
	assert.Equal(t, ErrNoMoreBytes, err)
	assert.Equal(t, uint16(0), v)
}

func Test_ReadNetUint16Failed(t *testing.T) {
	b := New(nil)
	b.SetError(ErrTest)
	v, err := b.ReadNetUint16()
	assert.Equal(t, ErrTest, err)
	assert.Equal(t, uint16(0), v)
}

func Test_ReadNetUint16Data(t *testing.T) {
	b := New([]byte{0xf1, 0xf2})
	v, err := b.ReadNetUint16()
	b.checkInvariants(t)
	assert.Nil(t, err)
	assert.Equal(t, uint16(0xf1f2), v)
}

func Test_ReadNetUint16AtFailed(t *testing.T) {
	b := New(nil)
	b.SetError(ErrTest)
	v, err := b.ReadNetUint16At(4)
	assert.Equal(t, ErrTest, err)
	assert.Equal(t, uint16(0), v)
}

func Test_ReadNetUint16AtInRange(t *testing.T) {
	b := New([]byte{0xf1, 0xf2, 0xf3})
	v, err := b.ReadNetUint16At(1)
	assert.Nil(t, err)
	assert.Equal(t, uint16(0xf2f3), v)
}

func Test_ReadNetUint16AtOutOfRange(t *testing.T) {
	b := New([]byte{0xf1, 0xf2, 0xf3})
	v, err := b.ReadNetUint16At(2)
	assert.Equal(t, ErrNoMoreBytes, err)
	assert.Equal(t, uint16(0), v)
}

func Test_WriteNetUint16At(t *testing.T) {
	b := New(nil)
	err := b.WriteNetUint16At(0x1f2f, 1)
	assert.Nil(t, err)

	b.Advance(1)
	tmp, err := b.ReadNetUint16()
	assert.Nil(t, err)
	assert.Equal(t, uint16(0x1f2f), tmp)
}

func Test_ReadNetUint32NoData(t *testing.T) {
	b := New(nil)
	v, err := b.ReadNetUint32()
	assert.Equal(t, ErrNoMoreBytes, err)
	assert.Equal(t, uint32(0), v)
}

func Test_ReadNetUint32Failed(t *testing.T) {
	b := New(nil)
	b.SetError(ErrTest)
	v, err := b.ReadNetUint32()
	assert.Equal(t, ErrTest, err)
	assert.Equal(t, uint32(0), v)
}

func Test_ReadNetUint32Data(t *testing.T) {
	b := New([]byte{0xf1, 0xf2, 0xf3, 0xf4})
	v, err := b.ReadNetUint32()
	b.checkInvariants(t)
	assert.Nil(t, err)
	assert.Equal(t, uint32(0xf1f2f3f4), v)
}

func Test_ReadNetUint32AtFailed(t *testing.T) {
	b := New(nil)
	b.SetError(ErrTest)
	v, err := b.ReadNetUint32At(4)
	assert.Equal(t, ErrTest, err)
	assert.Equal(t, uint32(0), v)
}

func Test_ReadNetUint32AtInRange(t *testing.T) {
	b := New([]byte{0xf1, 0xf2, 0xf3, 0xf4, 0xf5})
	v, err := b.ReadNetUint32At(1)
	assert.Nil(t, err)
	assert.Equal(t, uint32(0xf2f3f4f5), v)
}

func Test_ReadNetUint32AtOutOfRange(t *testing.T) {
	b := New([]byte{0xf1, 0xf2, 0xf3})
	v, err := b.ReadNetUint32At(2)
	assert.Equal(t, ErrNoMoreBytes, err)
	assert.Equal(t, uint32(0), v)
}

func Test_WriteNetUint32At(t *testing.T) {
	b := New(nil)
	err := b.WriteNetUint32At(0x1f2f3f4f, 1)
	assert.Nil(t, err)

	b.Advance(1)
	tmp, err := b.ReadNetUint32()
	assert.Nil(t, err)
	assert.Equal(t, uint32(0x1f2f3f4f), tmp)
}

func Test_ReadNetUint64NoData(t *testing.T) {
	b := New(nil)
	v, err := b.ReadNetUint64()
	assert.Equal(t, ErrNoMoreBytes, err)
	assert.Equal(t, uint64(0), v)
}

func Test_ReadNetUint64Failed(t *testing.T) {
	b := New(nil)
	b.SetError(ErrTest)
	v, err := b.ReadNetUint64()
	assert.Equal(t, ErrTest, err)
	assert.Equal(t, uint64(0), v)
}

func Test_ReadNetUint64Data(t *testing.T) {
	b := New([]byte{
		0xf0, 0xf1, 0xf2, 0xf3, 0xf4,
		0xf5, 0xf6, 0xf7, 0xf8, 0xf9,
		0xfa, 0xfb, 0xfc, 0xfd, 0xfe,
	})
	v, err := b.ReadNetUint64()
	b.checkInvariants(t)
	assert.Nil(t, err)
	assert.Equal(t, uint64(0xf0f1f2f3f4f5f6f7), v)
}

func Test_ReadNetUint64AtFailed(t *testing.T) {
	b := New(nil)
	b.SetError(ErrTest)
	v, err := b.ReadNetUint64At(4)
	assert.Equal(t, ErrTest, err)
	assert.Equal(t, uint64(0), v)
}

func Test_ReadNetUint64AtInRange(t *testing.T) {
	b := New([]byte{
		0xf0, 0xf1, 0xf2, 0xf3, 0xf4,
		0xf5, 0xf6, 0xf7, 0xf8, 0xf9,
		0xfa, 0xfb, 0xfc, 0xfd, 0xfe,
	})
	v, err := b.ReadNetUint64At(1)
	assert.Nil(t, err)
	assert.Equal(t, uint64(0xf1f2f3f4f5f6f7f8), v)
}

func Test_ReadNetUint64AtOutOfRange(t *testing.T) {
	b := New([]byte{0xf1, 0xf2, 0xf3})
	v, err := b.ReadNetUint64At(2)
	assert.Equal(t, ErrNoMoreBytes, err)
	assert.Equal(t, uint64(0), v)
}

func Test_WriteNetUint64At(t *testing.T) {
	b := New(nil)
	err := b.WriteNetUint64At(0x1f2f3f4f5f6f7f8f, 1)
	assert.Nil(t, err)

	b.Advance(1)
	tmp, err := b.ReadNetUint64()
	assert.Nil(t, err)
	assert.Equal(t, uint64(0x1f2f3f4f5f6f7f8f), tmp)
}
