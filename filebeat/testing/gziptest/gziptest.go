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

package gziptest

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/require"
)

type Corruption int

const (
	CorruptNone Corruption = 0
	CorruptCRC  Corruption = 1 << iota
	CorruptSize
)

// Compress takes input data, compresses it using gzip, and then, if specified,
// intentionally corrupts parts of the footer (CRC32 and/or ISIZE) to simulate
// checksum/length errors upon decompression.
// It returns the compressed GZIP data.
// Check the RFC1 952 for details https://www.rfc-editor.org/rfc/rfc1952.html.
func Compress(t *testing.T, data []byte, corruption Corruption) []byte {
	var gzBuff bytes.Buffer
	gw := gzip.NewWriter(&gzBuff)

	wrote, err := gw.Write(data)
	require.NoError(t, err, "failed to write data to gzip writer")

	// sanity check
	require.Equal(t, len(data), wrote, "written data is not equal to input data")
	require.NoError(t, gw.Close(), "failed to close gzip writer")

	compressedBytes := gzBuff.Bytes()

	footerStartIndex := len(compressedBytes) - 8

	if corruption&CorruptCRC != 0 {
		// CRC32 - first 4 bytes of footer
		originalCRC32 := binary.LittleEndian.Uint32(
			compressedBytes[footerStartIndex : footerStartIndex+4])
		// corrupted the CRC32, anything will do.
		corruptedCRC32 := originalCRC32 + 1
		binary.LittleEndian.PutUint32(
			compressedBytes[footerStartIndex:footerStartIndex+4], corruptedCRC32)
	}

	if corruption&CorruptSize != 0 {
		// ISIZE - last 4 bytes of footer
		originalISIZE := binary.LittleEndian.Uint32(
			compressedBytes[footerStartIndex+4 : footerStartIndex+8])
		// corrupted the ISIZE, anything will do
		corruptedISIZE := originalISIZE + 1
		binary.LittleEndian.PutUint32(
			compressedBytes[footerStartIndex+4:footerStartIndex+8], corruptedISIZE)
	}

	return compressedBytes
}
