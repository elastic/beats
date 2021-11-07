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

package auparse

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const nullTerminator = "\x00"

func hexToString(h string) (string, error) {
	output, err := decodeUppercaseHexString(h)
	if err != nil {
		return "", err
	}

	nullTerm := bytes.Index(output, []byte(nullTerminator))
	if nullTerm != -1 {
		output = output[:nullTerm]
	}

	return string(output), nil
}

func hexToStrings(h string) ([]string, error) {
	output, err := decodeUppercaseHexString(h)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(output), nullTerminator), nil
}

func hexToDec(h string) (int32, error) {
	num, err := strconv.ParseInt(h, 16, 32)
	return int32(num), err
}

func hexToIP(h string) (string, error) {
	if len(h) == 8 {
		a1, _ := hexToDec(h[0:2])
		a2, _ := hexToDec(h[2:4])
		a3, _ := hexToDec(h[4:6])
		a4, _ := hexToDec(h[6:8])
		return fmt.Sprintf("%d.%d.%d.%d", a1, a2, a3, a4), nil
	} else if len(h) == 32 {
		b, err := hex.DecodeString(h)
		if err != nil {
			return "", err
		}
		return net.IP(b).String(), nil
	}

	return "", errors.New("invalid size")
}

// decodeUppercaseHex decodes src into hex.DecodedLen(len(src)) bytes,
// returning the actual number of bytes written to dst.
//
// Decode expects that src contain only hexadecimal
// characters and that src should have an even length.
func decodeUppercaseHex(dst, src []byte) (int, error) {
	if len(src)%2 == 1 {
		return 0, hex.ErrLength
	}

	for i := 0; i < len(src)/2; i++ {
		a, ok := fromHexChar(src[i*2])
		if !ok {
			return 0, hex.InvalidByteError(src[i*2])
		}
		b, ok := fromHexChar(src[i*2+1])
		if !ok {
			return 0, hex.InvalidByteError(src[i*2+1])
		}
		dst[i] = (a << 4) | b
	}

	return len(src) / 2, nil
}

// fromHexChar converts a hex character into its value and a success flag.
func fromHexChar(c byte) (byte, bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}

	return 0, false
}

// decodeUppercaseHexString returns the bytes represented by the hexadecimal
// string s.
func decodeUppercaseHexString(s string) ([]byte, error) {
	src := []byte(s)
	dst := make([]byte, hex.DecodedLen(len(src)))
	_, err := decodeUppercaseHex(dst, src)
	if err != nil {
		return nil, err
	}
	return dst, nil
}
