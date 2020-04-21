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

package common

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	// 0xd800-0xdc00 encodes the high 10 bits of a pair.
	// 0xdc00-0xe000 encodes the low 10 bits of a pair.
	// the value is those 20 bits plus 0x10000.
	surr1           = 0xd800
	surr2           = 0xdc00
	surr3           = 0xe000
	replacementChar = '\uFFFD' // Unicode replacement character
)

// Byte order utilities

func BytesNtohs(b []byte) uint16 {
	return uint16(b[0])<<8 | uint16(b[1])
}

func BytesNtohl(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 |
		uint32(b[2])<<8 | uint32(b[3])
}

func BytesHtohl(b []byte) uint32 {
	return uint32(b[3])<<24 | uint32(b[2])<<16 |
		uint32(b[1])<<8 | uint32(b[0])
}

func BytesNtohll(b []byte) uint64 {
	return uint64(b[0])<<56 | uint64(b[1])<<48 |
		uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 |
		uint64(b[6])<<8 | uint64(b[7])
}

// Ipv4_Ntoa transforms an IP4 address in it's dotted notation
func IPv4Ntoa(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		byte(ip>>24), byte(ip>>16),
		byte(ip>>8), byte(ip))
}

// ReadString extracts the first null terminated string from
// a slice of bytes.
func ReadString(s []byte) (string, error) {
	i := bytes.IndexByte(s, 0)
	if i < 0 {
		return "", errors.New("No string found")
	}
	res := string(s[:i])
	return res, nil
}

// RandomBytes return a slice of random bytes of the defined length
func RandomBytes(length int) ([]byte, error) {
	r := make([]byte, length)
	_, err := rand.Read(r)

	if err != nil {
		return nil, err
	}

	return r, nil
}

func UTF16ToUTF8Bytes(in []byte, out io.Writer) error {
	if len(in)%2 != 0 {
		return fmt.Errorf("input buffer must have an even length (length=%d)", len(in))
	}

	var runeBuf [4]byte
	var v1, v2 uint16
	for i := 0; i < len(in); i += 2 {
		v1 = uint16(in[i]) | uint16(in[i+1])<<8
		// Stop at null-terminator.
		if v1 == 0 {
			return nil
		}

		switch {
		case v1 < surr1, surr3 <= v1:
			n := utf8.EncodeRune(runeBuf[:], rune(v1))
			out.Write(runeBuf[:n])
		case surr1 <= v1 && v1 < surr2 && len(in) > i+2:
			v2 = uint16(in[i+2]) | uint16(in[i+3])<<8
			if surr2 <= v2 && v2 < surr3 {
				// valid surrogate sequence
				r := utf16.DecodeRune(rune(v1), rune(v2))
				n := utf8.EncodeRune(runeBuf[:], r)
				out.Write(runeBuf[:n])
			}
			i += 2
		default:
			// invalid surrogate sequence
			n := utf8.EncodeRune(runeBuf[:], replacementChar)
			out.Write(runeBuf[:n])
		}
	}
	return nil
}

func StringToUTF16Bytes(in string) []byte {
	var u16 []uint16 = utf16.Encode([]rune(in))
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.LittleEndian, u16)
	return buf.Bytes()
}
