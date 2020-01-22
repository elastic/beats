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

package cborl

import "github.com/elastic/go-structform/internal/unsafe"

const (
	majorUint  uint8 = 0x00
	majorNeg   uint8 = 1 << 5
	majorBytes uint8 = 2 << 5
	majorText  uint8 = 3 << 5
	majorArr   uint8 = 4 << 5
	majorMap   uint8 = 5 << 5
	majorTag   uint8 = 6 << 5
	majorOther uint8 = 7 << 5

	majorMask uint8 = 7 << 5
	minorMask uint8 = ^majorMask
)

const (
	lenSmall uint8 = 0
	len8b    uint8 = 24
	len16b   uint8 = 25
	len32b   uint8 = 26
	len64b   uint8 = 27
	lenIndef uint8 = 31
)

const (
	codeFalse uint8 = 20 | majorOther
	codeTrue  uint8 = 21 | majorOther
	codeNull  uint8 = 22 | majorOther
	codeUndef uint8 = 23 | majorOther

	codeHalfFloat   uint8 = 25 | majorOther
	codeSingleFloat uint8 = 26 | majorOther
	codeDoubleFloat uint8 = 27 | majorOther
	codeBreak       uint8 = lenIndef | majorOther
)

func str2Bytes(s string) []byte {
	return unsafe.Str2Bytes(s)
}

func bytes2Str(b []byte) string {
	return unsafe.Bytes2Str(b)
}
