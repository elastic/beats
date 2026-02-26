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

package nfs

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testMsg = []byte{
	0x80, 0x00, 0x00, 0xe0,
	0xb5, 0x49, 0x21, 0xab,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
	0x00, 0x00, 0x00, 0x04,

	0x00, 0x00, 0x00, 0x0b,
	0x74, 0x65, 0x73, 0x74, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x00,
}

func BytesToUint32(b []byte) []uint32 {
	n := len(b) / 4
	out := make([]uint32, n)
	for i := 0; i < n; i++ {
		out[i] = binary.BigEndian.Uint32(b[i*4:])
	}
	return out
}

func TestXdrDecoding(t *testing.T) {
	xdr := makeXDR(testMsg)

	v, err := xdr.getUInt()
	require.NoError(t, err)
	assert.Equal(t, uint32(0x800000e0), v)

	v, err = xdr.getUInt()
	require.NoError(t, err)
	assert.Equal(t, uint32(0xb54921ab), v)

	hv, err := xdr.getUHyper()
	require.NoError(t, err)
	assert.Equal(t, uint64(2), hv)

	v, err = xdr.getUInt()
	require.NoError(t, err)
	assert.Equal(t, uint32(4), v)

	str, err := xdr.getString()
	require.NoError(t, err)
	assert.Equal(t, "test string", str)
	assert.Equal(t, len(testMsg), xdr.size())
}

func TestXdrDecodingFailures(t *testing.T) {

	// test getUInt
	xdr := makeXDR([]byte{0x80})
	_, err := xdr.getUInt()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x80, 0x00})
	_, err = xdr.getUInt()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x80, 0x00, 0x00})
	_, err = xdr.getUInt()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x80, 0x00, 0x00, 0x01})
	v, err := xdr.getUInt()
	require.NoError(t, err)
	assert.Equal(t, uint32(0x80000001), v)

	// test getUHyper
	xdr = makeXDR([]byte{0x80})
	_, err = xdr.getUHyper()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x80, 0x00})
	_, err = xdr.getUHyper()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x80, 0x00, 0x00})
	_, err = xdr.getUHyper()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x80, 0x00, 0x00, 0x01})
	_, err = xdr.getUHyper()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x80, 0x00, 0x00, 0x01, 0x80})
	_, err = xdr.getUHyper()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x80, 0x00, 0x00, 0x01, 0x80, 0x00})
	_, err = xdr.getUHyper()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x80, 0x00, 0x00, 0x01, 0x80, 0x00, 0x00})
	_, err = xdr.getUHyper()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x80, 0x00, 0x00, 0x01, 0x80, 0x00, 0x00, 0x01})
	v64, err := xdr.getUHyper()
	require.NoError(t, err)
	assert.Equal(t, uint64(0x8000000180000001), v64)

	// test getString
	xdr = makeXDR([]byte{0x00})
	_, err = xdr.getString()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x00, 0x00})
	_, err = xdr.getString()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x00, 0x00, 0x00})
	_, err = xdr.getString()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x00, 0x00, 0x00, 0x03})
	_, err = xdr.getString()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x00, 0x00, 0x00, 0x03, 0x68})
	_, err = xdr.getString()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x00, 0x00, 0x00, 0x03, 0x68, 0x69})
	_, err = xdr.getString()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x00, 0x00, 0x00, 0x03, 0x68, 0x69, 0x21})
	_, err = xdr.getString()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x00, 0x00, 0x00, 0x03, 0x68, 0x69, 0x21, 0x00})
	msg, err := xdr.getString()
	require.NoError(t, err)
	assert.Equal(t, "hi!", msg)

	// test getOpaque
	b := make([]byte, maxOpaque+1)
	xdr = makeXDR(b)
	_, err = xdr.getOpaque(maxOpaque + 1)
	require.Error(t, err)

	b = make([]byte, maxOpaque+1)
	xdr = makeXDR(b)
	_, err = xdr.getOpaque(-1)
	require.Error(t, err)

	b = make([]byte, maxOpaque-1)
	xdr = makeXDR(b)
	_, err = xdr.getOpaque(maxOpaque)
	require.Error(t, err)

	b = make([]byte, maxOpaque)
	xdr = makeXDR(b)
	ba, err := xdr.getOpaque(maxOpaque)
	require.NoError(t, err)
	assert.Equal(t, maxOpaque, len(ba))

	xdr = makeXDR([]byte{0x80})
	_, err = xdr.getOpaque(1)
	require.Error(t, err)

	xdr = makeXDR([]byte{0x80, 0x00})
	_, err = xdr.getOpaque(1)
	require.Error(t, err)

	xdr = makeXDR([]byte{0x80, 0x00, 0x00})
	_, err = xdr.getOpaque(1)
	require.Error(t, err)

	xdr = makeXDR([]byte{0x80, 0x00, 0x00, 0x00})
	ba, err = xdr.getOpaque(1)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x80}, ba)

	// test getDynamicOpaque
	xdr = makeXDR([]byte{0x00})
	_, err = xdr.getDynamicOpaque()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x00, 0x00})
	_, err = xdr.getDynamicOpaque()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x00, 0x00, 0x00})
	_, err = xdr.getDynamicOpaque()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x00, 0x00, 0x00, 0x01})
	_, err = xdr.getDynamicOpaque()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x00, 0x00, 0x00, 0x01, 0x08})
	_, err = xdr.getDynamicOpaque()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x00, 0x00, 0x00, 0x01, 0x08, 0x00})
	_, err = xdr.getDynamicOpaque()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x00, 0x00, 0x00, 0x01, 0x08, 0x00, 0x00})
	_, err = xdr.getDynamicOpaque()
	require.Error(t, err)

	xdr = makeXDR([]byte{0x00, 0x00, 0x00, 0x01, 0x08, 0x00, 0x00, 0x00})
	ba, err = xdr.getDynamicOpaque()
	require.NoError(t, err)
	assert.Equal(t, []byte{0x08}, ba)

	// test getUintVector
	length := []byte{0x00, 0x00, 0x80, 0x00} // maxVector
	uv := make([]byte, maxVector*4)
	data := append(length, uv...)
	xdr = makeXDR(data)
	uva, err := xdr.getUIntVector()
	require.NoError(t, err)
	assert.Equal(t, BytesToUint32(uv), uva)

	// test length longer than buffer
	length = []byte{0x00, 0x00, 0x80, 0x00} // maxVector
	uv = make([]byte, (maxVector-1)*4)
	data = append(length, uv...)
	xdr = makeXDR(data)
	_, err = xdr.getUIntVector()
	require.Error(t, err)

	// test matching length and buffer exceed size limits
	length = []byte{0x00, 0x00, 0x80, 0x01} // maxVector + 1
	uv = make([]byte, (maxVector+1)*4)
	data = append(length, uv...)
	xdr = makeXDR(data)
	_, err = xdr.getUIntVector()
	require.Error(t, err)
}
