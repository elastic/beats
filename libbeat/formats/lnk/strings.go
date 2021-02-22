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
	"errors"
	"io"

	"github.com/elastic/beats/v7/libbeat/formats/common"
)

func readU16Data(offset int64, r io.ReaderAt, hasUnicode bool) (uint16, []byte, error) {
	sizeData := make([]byte, 2)
	n, err := r.ReadAt(sizeData, offset)
	if err != nil {
		return 0, nil, err
	}
	if n != 2 {
		return 0, nil, errors.New("invalid size")
	}
	size := binary.LittleEndian.Uint16(sizeData)
	if hasUnicode {
		size *= 2
	}
	data := make([]byte, size)
	n, err = r.ReadAt(data, offset+2)
	if uint16(n) != size {
		return 0, nil, errors.New("invalid data")
	}
	return size, data, nil
}

func readU32Data(offset int64, r io.ReaderAt) (uint32, []byte, error) {
	sizeData := make([]byte, 4)
	n, err := r.ReadAt(sizeData, offset)
	if err != nil {
		return 0, nil, err
	}
	if n != 4 {
		return 0, nil, errors.New("invalid size")
	}
	size := binary.LittleEndian.Uint32(sizeData)
	data := make([]byte, size)
	n, err = r.ReadAt(data, offset)
	if uint32(n) != size {
		return 0, nil, errors.New("invalid data")
	}
	return size, data, nil
}

func readDataString(header *Header, flag uint32, offset int64, r io.ReaderAt) (string, int64, error) {
	if !hasFlag(header.rawLinkFlags, flag) {
		return "", offset, nil
	}
	hasUnicode := hasFlag(header.rawLinkFlags, isUnicode)
	size, data, err := readU16Data(offset, r, hasUnicode)
	if err != nil {
		return "", 0, err
	}
	if hasUnicode {
		return common.ReadUnicode(data, 0), offset + 2 + int64(size), nil
	}
	return common.ReadString(data, 0), offset + 2 + int64(size), nil
}
