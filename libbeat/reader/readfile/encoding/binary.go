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

package encoding

import (
	g_binary "encoding/binary"
	"errors"
	"fmt"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

var ErrNotEnoughBytes = errors.New("not enough data in slice")

type BinaryDecodeError struct {
	msg string
}

func (d BinaryDecodeError) Error() string {
	return d.msg
}

func NewBinaryDecodeError(msg string) BinaryDecodeError {
	return BinaryDecodeError{
		msg: msg,
	}
}

type BinaryConfigError struct {
	msg string
}

func (d BinaryConfigError) Error() string {
	return d.msg
}

func NewBinaryConfigError(msg string) BinaryConfigError {
	return BinaryConfigError{
		msg: msg,
	}
}

type BinaryEncoding struct {
	Enabled bool `config:"enabled"`
	Header int `config:"header-length"`
	Offset int `config:"offset"`
	Length int `config:"length"`
	ByteOrder  string `config:"byte-order"`
	order g_binary.ByteOrder
}

func (b *BinaryEncoding) Validate() error {

	var errors []string
	if b.Header < 0 {
		errors = append(errors, fmt.Sprintf("invalid header! [%d] must be non-negative",
			b.Header))
	}

	if b.Length < 1 {
		errors = append(errors, fmt.Sprintf("invalid length! [%d] must be positive",
			b.Length))
	}

	if b.Offset < 0 {
		errors = append(errors, fmt.Sprintf("invalid offset! [%d] must be non-negative",
			b.Offset))
	}

	switch b.ByteOrder {
	case "big-endian":
		b.order = g_binary.BigEndian
	case "bigendian":
		b.order = g_binary.BigEndian
	case "big":
		b.order = g_binary.BigEndian
	case "little-endian":
		b.order = g_binary.LittleEndian
	case "littleendian":
		b.order = g_binary.LittleEndian
	case "little":
		b.order = g_binary.LittleEndian

	case "native-endian":
		b.order = g_binary.NativeEndian
	case "nativeendian":
		b.order = g_binary.NativeEndian
	case "native":
		b.order = g_binary.NativeEndian

	default:
		// should this be an error?
		b.order = g_binary.NativeEndian
	}

	if len(errors) > 0 {
		result := NewBinaryConfigError(fmt.Sprintf("Invalid Config! %v", errors))
		// if it's disabled, don't care?
		if b.Enabled == false {
			fmt.Printf("ignoring error %v\n", result.Error())
			return nil
		}
		return result
	}

	return nil
}

func DefaultBinaryEncoding() BinaryEncoding {
	return BinaryEncoding {
		Enabled: false,
		Header: 0,
		Offset: 0,
		Length: 0,
		ByteOrder: "",
		order: g_binary.NativeEndian,
	}
}

func (b BinaryEncoding) MinimumLength() int {
	if b.Header > 0 {
		return b.Header
	}
	return b.Offset + b.Length
}

func (b BinaryEncoding) GetMessageLength(data []byte) (int, error) {
	var result int

	if len(data) < b.Offset + b.Length {
		return 0, ErrNotEnoughBytes
	}

	msg := data[b.Offset:b.Offset+b.Length]
	result, err := b.DecodeBytes(msg)
	if err != nil {
		return result, err
	}

	if b.Header > 0 {
		result += b.Header
	}
	return result, nil
}


func (b BinaryEncoding) decode8(buf []byte) int {
	return int(uint8(buf[0]))
}
func (b BinaryEncoding) decode16(buf []byte) int {
	return int(b.order.Uint16(buf[0:2]))
}
func (b BinaryEncoding) decode32(buf []byte) int {
	return int(b.order.Uint32(buf[0:4]))
}
func (b BinaryEncoding) DecodeBytes(data []byte) (int, error) {
	if len(data) < b.Length {
		return 0, ErrNotEnoughBytes
	}

	switch b.Length {
	case 1:
		return b.decode8(data), nil
	case 2:
		return b.decode16(data), nil
	case 4:
		return b.decode32(data), nil
	default:
		return 0, NewBinaryDecodeError(fmt.Sprintf("Invalid lenght: [%d] should be 1, 2, 4", b.Length))
	}
}

type binenc struct {}

// func (b binary) Reset() {}
// func (b binary) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) { }

func (binenc) NewDecoder() *encoding.Decoder {
	fmt.Println("Creating new Binary Decoder")
	return &encoding.Decoder{
		Transformer: transform.Nop,
	}
}

func (binenc) NewEncoder() *encoding.Encoder {
	fmt.Println("Creating new Binary Encoder")
	return &encoding.Encoder{
		Transformer: transform.Nop,
	}
}
