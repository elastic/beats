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

// +build !plan9,!openbsd

package readfile

import (
	"fmt"
	"unsafe"
)

// alignment returns alignment of the block in memory
// with reference to AlignSize
//
// Can't check alignment of a zero sized block as &block[0] is invalid
func alignment(block []byte, AlignSize int) int {
	return int(uintptr(unsafe.Pointer(&block[0])) & uintptr(AlignSize-1))
}

// AlignedBlock returns []byte of size BlockSize aligned to a multiple
// of AlignSize in memory (must be power of two)
func AlignedBlock(blockSize int) ([]byte, error) {
	if blockSize < MinimumBlockSize {
		blockSize = MinimumBlockSize
	}

	block := make([]byte, blockSize+AlignSize)
	if AlignSize == 0 {
		return block, nil
	}
	a := alignment(block, AlignSize)
	offset := 0
	if a != 0 {
		offset = AlignSize - a
	}
	block = block[offset : offset+blockSize]
	// Can't check alignment of a zero sized block
	if blockSize != 0 {
		a = alignment(block, AlignSize)
		if a != 0 {
			return nil, fmt.Errorf("Failed to align block")
		}
	}
	return block, nil
}

// MakeBuffer returns []byte of bufferSize and aligns it if needed
// Align works only on all systems except OpenBSD and Plan9
func MakeBuffer(bufferSize int, align bool) ([]byte, error) {
	if align {
		return AlignedBlock(bufferSize)
	}

	return make([]byte, bufferSize), nil
}
