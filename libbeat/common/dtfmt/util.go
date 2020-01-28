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
	"strconv"
)

func appendUnpadded(bs []byte, i int) []byte {
	return strconv.AppendInt(bs, int64(i), 10)
}

func appendPadded(bs []byte, val, digits int) []byte {
	if val < 0 {
		bs = append(bs, '-')
		val = -1
	}

	// compute number of initial padding zeroes
	var padDigits int
	switch {
	case val < 10:
		padDigits = digits - 1
	case val < 100:
		padDigits = digits - 2
	case val < 1000:
		padDigits = digits - 3
	case val < 10000:
		padDigits = digits - 4
	case val < 100000:
		padDigits = digits - 5
	case val < 1000000:
		padDigits = digits - 6
	case val < 10000000:
		padDigits = digits - 7
	case val < 100000000:
		padDigits = digits - 8
	case val < 1000000000:
		padDigits = digits - 9
	default:
		padDigits = digits - 1
		for tmp := val; tmp > 10; tmp = tmp / 10 {
			padDigits--
		}
	}
	for i := 0; i < padDigits; i++ {
		bs = append(bs, '0')
	}

	// encode value
	if val < 10 {
		return append(bs, byte(val)+'0')
	}
	return strconv.AppendInt(bs, int64(val), 10)
}

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
	begin := initalLen + ((digits-1)/fractSz)*fractSz
	end := begin + fractSz
	if L := len(bs); end >= L {
		end = L
	}

	// remove trailing zeros, such that a multiple of fractSz digits will be
	// presend in the final buffer. At minimum fractSz digits will always be
	// reported.
	for {
		if !allZero(bs[begin:end]) {
			break
		}

		digits -= (end - begin) // invariant: 0 <= end - being <= fractSz
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
}
