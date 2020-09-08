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

package diskqueue

import (
	"encoding/binary"
	"hash/crc32"
)

// ChecksumType specifies what checksum algorithm the queue should use to
// verify its data frames.
type ChecksumType uint32

// ChecksumTypeNone: Don't compute or verify checksums.
// ChecksumTypeCRC32: Compute the checksum with the Go standard library's
//   "hash/crc32" package.
const (
	ChecksumTypeNone = iota

	ChecksumTypeCRC32
)

func computeChecksum(data []byte, checksumType ChecksumType) uint32 {
	switch checksumType {
	case ChecksumTypeNone:
		return 0
	case ChecksumTypeCRC32:
		hash := crc32.NewIEEE()
		frameLength := uint32(len(data) + frameMetadataSize)
		binary.Write(hash, binary.LittleEndian, &frameLength)
		hash.Write(data)
		return hash.Sum32()
	default:
		panic("segmentReader: invalid checksum type")
	}
}
