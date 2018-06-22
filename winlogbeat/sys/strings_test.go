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

package sys

import (
	"bytes"
	"encoding/binary"
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/assert"
)

func toUTF16Bytes(in string) []byte {
	var u16 []uint16 = utf16.Encode([]rune(in))
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.LittleEndian, u16)
	return buf.Bytes()
}

func TestUTF16BytesToString(t *testing.T) {
	input := "abc白鵬翔\u145A6"
	utf16Bytes := toUTF16Bytes(input)

	output, _, err := UTF16BytesToString(utf16Bytes)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, input, output)
}

func TestUTF16BytesToStringOffset(t *testing.T) {
	in := bytes.Join([][]byte{toUTF16Bytes("one"), toUTF16Bytes("two"), toUTF16Bytes("three")}, []byte{0, 0})

	output, offset, err := UTF16BytesToString(in)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "one", output)
	assert.Equal(t, 8, offset)

	in = in[offset:]
	output, offset, err = UTF16BytesToString(in)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "two", output)
	assert.Equal(t, 8, offset)

	in = in[offset:]
	output, offset, err = UTF16BytesToString(in)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "three", output)
	assert.Equal(t, -1, offset)
}

func TestUTF16BytesToStringOffsetWithEmptyString(t *testing.T) {
	in := bytes.Join([][]byte{toUTF16Bytes(""), toUTF16Bytes("two")}, []byte{0, 0})

	output, offset, err := UTF16BytesToString(in)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "", output)
	assert.Equal(t, 2, offset)

	in = in[offset:]
	output, offset, err = UTF16BytesToString(in)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "two", output)
	assert.Equal(t, -1, offset)
}

func BenchmarkUTF16BytesToString(b *testing.B) {
	utf16Bytes := toUTF16Bytes("A logon was attempted using explicit credentials.")

	b.Run("simple_string", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			UTF16BytesToString(utf16Bytes)
		}
	})

	// Buffer larger than the string.
	b.Run("larger_buffer", func(b *testing.B) {
		utf16Bytes = append(utf16Bytes, make([]byte, 2048)...)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			UTF16BytesToString(utf16Bytes)
		}
	})
}

func TestUTF16ToUTF8(t *testing.T) {
	input := "abc白鵬翔\u145A6"
	utf16Bytes := toUTF16Bytes(input)

	outputBuf := &bytes.Buffer{}
	err := UTF16ToUTF8Bytes(utf16Bytes, outputBuf)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []byte(input), outputBuf.Bytes())
}

func TestUTF16BytesToStringTrimNullTerm(t *testing.T) {
	input := "abc"
	utf16Bytes := append(toUTF16Bytes(input), []byte{0, 0, 0, 0, 0, 0}...)

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
	utf16Bytes := toUTF16Bytes("A logon was attempted using explicit credentials.")
	outputBuf := &bytes.Buffer{}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		UTF16ToUTF8Bytes(utf16Bytes, outputBuf)
		outputBuf.Reset()
	}
}
