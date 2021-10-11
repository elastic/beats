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

package tls

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBufferView(t *testing.T) {
	buf := mkBuf(t, "0123456789abcdef", 8)
	var (
		u8  uint8
		u16 uint16
		u32 uint32
	)
	assert.Equal(t, 8, buf.length())
	assert.True(t, buf.read8(0, &u8))
	assert.Equal(t, uint8(1), u8)

	assert.True(t, buf.read8(1, &u8))
	assert.Equal(t, uint8(0x23), u8)

	assert.True(t, buf.read16Net(1, &u16))
	assert.Equal(t, uint16(0x2345), u16)

	assert.True(t, buf.read16Net(2, &u16))
	assert.Equal(t, uint16(0x4567), u16)

	assert.True(t, buf.read16Net(6, &u16))
	assert.Equal(t, uint16(0xcdef), u16)

	assert.True(t, buf.read24Net(3, &u32))
	assert.Equal(t, uint32(0x6789ab), u32)

	assert.True(t, buf.read24Net(4, &u32))
	assert.Equal(t, uint32(0x89abcd), u32)

	assert.True(t, buf.read24Net(5, &u32))
	assert.Equal(t, uint32(0xabcdef), u32)

	assert.True(t, buf.read32Net(3, &u32))
	assert.Equal(t, uint32(0x6789abcd), u32)

	assert.True(t, buf.read32Net(4, &u32))
	assert.Equal(t, uint32(0x89abcdef), u32)
}

func TestLimit(t *testing.T) {
	buf := mkBuf(t, "0123456789abcdef", 8)
	var (
		u8  uint8
		u16 uint16
		u32 uint32
	)
	assert.Equal(t, 8, buf.length())
	assert.True(t, buf.read8(7, &u8))
	assert.Equal(t, uint8(0xef), u8)

	assert.False(t, buf.read8(8, &u8))
	assert.False(t, buf.read16Net(7, &u16))
	assert.False(t, buf.read24Net(6, &u32))
	assert.False(t, buf.read32Net(5, &u32))

	assert.False(t, buf.read16Net(8, &u16))
	assert.False(t, buf.read24Net(7, &u32))
	assert.False(t, buf.read32Net(6, &u32))

	assert.False(t, buf.read24Net(8, &u32))
	assert.False(t, buf.read32Net(7, &u32))

	assert.False(t, buf.read32Net(8, &u32))
}

func TestError(t *testing.T) {
	buf := mkBuf(t, "010203", 8)
	var (
		u8  uint8
		u16 uint16
		u32 uint32
	)
	assert.Equal(t, 8, buf.length())
	assert.False(t, buf.read8(3, &u8))
	assert.False(t, buf.read16Net(2, &u16))
	assert.False(t, buf.read16Net(3, &u16))
	assert.False(t, buf.read24Net(1, &u32))
	assert.False(t, buf.read24Net(2, &u32))
	assert.False(t, buf.read24Net(3, &u32))

	assert.False(t, buf.read32Net(0, &u32))
	assert.False(t, buf.read32Net(1, &u32))
	assert.False(t, buf.read32Net(2, &u32))
	assert.False(t, buf.read32Net(3, &u32))
}

func TestString(t *testing.T) {
	buf := mkBuf(t, "313233", 3)
	var s string

	assert.True(t, buf.readString(0, 3, &s))
	assert.Equal(t, "123", s)
	assert.False(t, buf.readString(1, 5, &s))
}
