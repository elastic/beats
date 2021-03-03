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

package lnk

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"

	sha256 "github.com/minio/sha256-simd"
)

func parseShellbags(header *Header, offset int64, r io.ReaderAt) ([]Shellbag, int64, error) {
	if !hasFlag(header.rawLinkFlags, hasTargetIDList) {
		return nil, offset, nil
	}

	sizeData := make([]byte, 2)
	n, err := r.ReadAt(sizeData, offset)
	if err != nil {
		return nil, 0, err
	}
	if n != 2 {
		return nil, 0, errors.New("invalid target list")
	}
	offset += 2
	size := binary.LittleEndian.Uint16(sizeData)
	data := make([]byte, size)
	n, err = r.ReadAt(data, offset)
	if err != nil {
		return nil, 0, err
	}
	if n != int(size) {
		return nil, 0, errors.New("invalid target list size")
	}
	shellbags, err := parseShellbagList(data)
	return shellbags, offset + int64(size), err
}

func parseShellbagList(data []byte) ([]Shellbag, error) {
	// https://github.com/libyal/libfwsi/blob/master/documentation/Windows%20Shell%20Item%20format.asciidoc#2-shell-item-list
	shellbags := []Shellbag{}
	offset := 0
	for {
		shellbagData := data[offset:]
		if len(shellbagData) < 3 {
			// early end
			return shellbags, nil
		}
		shellbagSize := binary.LittleEndian.Uint16(shellbagData[0:2])
		if shellbagSize == 0 {
			return shellbags, nil
		}
		if len(shellbagData) < int(shellbagSize) {
			// we have an invalid target
			return shellbags, nil
		}
		shellbagData = shellbagData[:shellbagSize]
		shellbagType := shellbagData[2]
		hash := sha256.Sum256(shellbagData[3:])
		shellbags = append(shellbags, Shellbag{
			Name:   getShellbagName(shellbagType, shellbagData[3:]),
			Size:   shellbagSize,
			TypeID: shellbagType,
			SHA256: hex.EncodeToString(hash[:]),
		})
		offset += int(shellbagSize)
	}
}
