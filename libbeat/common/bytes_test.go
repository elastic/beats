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

package common

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/assert"
)

func TestBytes_Ntohs(t *testing.T) {
	type io struct {
		Input  []byte
		Output uint16
	}

	tests := []io{
		{
			Input:  []byte{0, 1},
			Output: 1,
		},
		{
			Input:  []byte{1, 0},
			Output: 256,
		},
		{
			Input:  []byte{1, 2},
			Output: 258,
		},
		{
			Input:  []byte{2, 3},
			Output: 515,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Output, BytesNtohs(test.Input))
	}
}

func TestBytes_Ntohl(t *testing.T) {
	type io struct {
		Input  []byte
		Output uint32
	}

	tests := []io{
		{
			Input:  []byte{0, 0, 0, 1},
			Output: 1,
		},
		{
			Input:  []byte{0, 0, 1, 0},
			Output: 256,
		},
		{
			Input:  []byte{0, 1, 0, 0},
			Output: 1 << 16,
		},
		{
			Input:  []byte{1, 0, 0, 0},
			Output: 1 << 24,
		},
		{
			Input:  []byte{1, 0, 15, 0},
			Output: 0x01000f00,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Output, BytesNtohl(test.Input))
	}
}

func TestBytes_Htohl(t *testing.T) {
	type io struct {
		Input  []byte
		Output uint32
	}

	tests := []io{
		{
			Input:  []byte{0, 0, 0, 1},
			Output: 1 << 24,
		},
		{
			Input:  []byte{0, 0, 1, 0},
			Output: 1 << 16,
		},
		{
			Input:  []byte{0, 1, 0, 0},
			Output: 256,
		},
		{
			Input:  []byte{1, 0, 0, 0},
			Output: 1,
		},
		{
			Input:  []byte{1, 0, 15, 0},
			Output: 0x000f0001,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Output, BytesHtohl(test.Input))
	}
}

func TestBytes_Ntohll(t *testing.T) {
	type io struct {
		Input  []byte
		Output uint64
	}

	tests := []io{
		{
			Input:  []byte{0, 0, 0, 0, 0, 0, 0, 1},
			Output: 1,
		},
		{
			Input:  []byte{0, 0, 0, 0, 0, 0, 1, 0},
			Output: 256,
		},
		{
			Input:  []byte{0, 0, 0, 0, 0, 1, 0, 0},
			Output: 1 << 16,
		},
		{
			Input:  []byte{0, 0, 0, 0, 1, 0, 0, 0},
			Output: 1 << 24,
		},
		{
			Input:  []byte{0, 0, 0, 1, 0, 0, 0, 0},
			Output: 1 << 32,
		},
		{
			Input:  []byte{0, 0, 1, 0, 0, 0, 0, 0},
			Output: 1 << 40,
		},
		{
			Input:  []byte{0, 1, 0, 0, 0, 0, 0, 0},
			Output: 1 << 48,
		},
		{
			Input:  []byte{1, 0, 0, 0, 0, 0, 0, 0},
			Output: 1 << 56,
		},
		{
			Input:  []byte{0, 1, 0, 0, 1, 0, 15, 0},
			Output: 0x0001000001000f00,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Output, BytesNtohll(test.Input))
	}
}

func TestIpv4_Ntoa(t *testing.T) {
	type io struct {
		Input  uint32
		Output string
	}

	tests := []io{
		{
			Input:  0x7f000001,
			Output: "127.0.0.1",
		},
		{
			Input:  0xc0a80101,
			Output: "192.168.1.1",
		},
		{
			Input:  0,
			Output: "0.0.0.0",
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Output, IPv4Ntoa(test.Input))
	}
}

func TestReadString(t *testing.T) {
	type io struct {
		Input  []byte
		Output string
		Err    error
	}

	tests := []io{
		{
			Input:  []byte{'a', 'b', 'c', 0, 'd', 'e', 'f'},
			Output: "abc",
			Err:    nil,
		},
		{
			Input:  []byte{0},
			Output: "",
			Err:    nil,
		},
		{
			Input:  []byte{'a', 'b', 'c'},
			Output: "",
			Err:    errors.New("No string found"),
		},
		{
			Input:  []byte{},
			Output: "",
			Err:    errors.New("No string found"),
		},
	}

	for _, test := range tests {
		res, err := ReadString(test.Input)
		assert.Equal(t, test.Err, err)
		assert.Equal(t, test.Output, res)
	}
}

func TestRandomBytesLength(t *testing.T) {
	r1, _ := RandomBytes(5)
	assert.Equal(t, len(r1), 5)

	r2, _ := RandomBytes(4)
	assert.Equal(t, len(r2), 4)
	assert.NotEqual(t, string(r1[:]), string(r2[:]))
}

func TestRandomBytes(t *testing.T) {
	v1, err := RandomBytes(10)
	assert.NoError(t, err)
	v2, err := RandomBytes(10)
	assert.NoError(t, err)

	// unlikely to get 2 times the same results
	assert.False(t, bytes.Equal(v1, v2))
}

func TestUTF16ToUTF8(t *testing.T) {
	input := "abc白鵬翔\u145A6"
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.LittleEndian, utf16.Encode([]rune(input)))
	outputBuf := &bytes.Buffer{}
	err := UTF16ToUTF8Bytes(buf.Bytes(), outputBuf)
	assert.NoError(t, err)
	assert.Equal(t, []byte(input), outputBuf.Bytes())
}

func TestUTF16BytesToStringTrimNullTerm(t *testing.T) {
	input := "abc"
	utf16Bytes := append(StringToUTF16Bytes(input), []byte{0, 0, 0, 0, 0, 0}...)

	outputBuf := &bytes.Buffer{}
	err := UTF16ToUTF8Bytes(utf16Bytes, outputBuf)
	if err != nil {
		t.Fatal(err)
	}
	b := outputBuf.Bytes()
	assert.Len(t, b, 3)
	assert.Equal(t, input, string(b))
}

func BenchmarkUTF16ToUTF8(b *testing.B) {
	utf16Bytes := StringToUTF16Bytes("A logon was attempted using explicit credentials.")
	outputBuf := &bytes.Buffer{}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		UTF16ToUTF8Bytes(utf16Bytes, outputBuf)
		outputBuf.Reset()
	}
}
