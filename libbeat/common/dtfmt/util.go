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

package dtfmt

import (
	"math"
	"strconv"
)

func appendUnpadded(bs []byte, i int) []byte {
	return strconv.AppendInt(bs, int64(i), 10)
}

<<<<<<< HEAD
func appendPadded(bs []byte, i, sz int) []byte {
	if i < 0 {
=======
// appendPadded appends a number value as string to the buffer. The string will
// be prefixed with '0' in case the encoded string value is takes less then
// 'digits' bytes.
//
// for example:
//
//	appendPadded(..., 10, 5) -> 00010
//	appendPadded(..., 12345, 5) -> 12345
func appendPadded(bs []byte, val, digits int) []byte {
	if val < 0 {
>>>>>>> e7e6dacfca ([updatecli][githubrelease] Bump version to 1.19.5 (#34497))
		bs = append(bs, '-')
		i = -i
	}

	if i < 10 {
		for ; sz > 1; sz-- {
			bs = append(bs, '0')
		}
		return append(bs, byte(i)+'0')
	}
	if i < 100 {
		for ; sz > 2; sz-- {
			bs = append(bs, '0')
		}
		return strconv.AppendInt(bs, int64(i), 10)
	}

	digits := 0
	if i < 1000 {
		digits = 3
	} else if i < 10000 {
		digits = 4
	} else {
		digits = int(math.Log10(float64(i))) + 1
	}
	for ; sz > digits; sz-- {
		bs = append(bs, '0')
	}

<<<<<<< HEAD
	return strconv.AppendInt(bs, int64(i), 10)
=======
	// encode value
	if val < 10 {
		return append(bs, byte(val)+'0')
	}
	return strconv.AppendInt(bs, int64(val), 10)
}

// appendFractPadded appends a number value as string to the buffer.
// The string will be prefixed with '0' in case the value is smaller than
// a value that can be represented with 'digits'.
// Trailing zeroes at the end will be removed, such that only a multiple of fractSz
// digits will be printed. If the value is 0, a total of 'fractSz' zeros will
// be printed.
//
// for example:
//
//	appendFractPadded(..., 0, 9, 3) -> "000"
//	appendFractPadded(..., 123000, 9, 3) -> "000123"
//	appendFractPadded(..., 120000, 9, 3) -> "000120"
//	appendFractPadded(..., 120000010, 9, 3) -> "000120010"
//	appendFractPadded(..., 123456789, 6, 3) -> "123456"
func appendFractPadded(bs []byte, val, digits, fractSz int) []byte {
	if fractSz == 0 || digits <= fractSz {
		return appendPadded(bs, val, digits)
	}

	initalLen := len(bs)
	bs = appendPadded(bs, val, digits)

	// find and remove trailing zeroes, such that a multiple of fractSz is still
	// serialized

	// find index range of last digits in buffer, such that a multiple of fractSz
	// will be kept if the range of digits is removed.
	// invariant: 0 <= end - begin <= fractSz
	end := len(bs)
	digits = end - initalLen
	begin := initalLen + ((digits-1)/fractSz)*fractSz

	// remove trailing zeros, such that a multiple of fractSz digits will be
	// present in the final buffer. At minimum fractSz digits will always be
	// reported.
	for {
		if !allZero(bs[begin:end]) {
			break
		}

		digits -= (end - begin)
		end = begin
		begin -= fractSz

		if digits <= fractSz {
			break
		}
	}

	return bs[:end]
}

func allZero(buf []byte) bool {
	for _, b := range buf {
		if b != '0' {
			return false
		}
	}
	return true
>>>>>>> e7e6dacfca ([updatecli][githubrelease] Bump version to 1.19.5 (#34497))
}
