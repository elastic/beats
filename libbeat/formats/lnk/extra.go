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
	"fmt"
	"io"
)

const (
	environmentBlock uint32 = 0xa0000001 + iota
	consoleBlock
	trackerBlock
	consoleFEBlock
	specialFolderBlock
	darwinBlock
	iconEnvironmentBlock
	shimBlock
	propertyStoreBlock
	_
	knownFolderBlock
	vistaAndAboveIDListBlock
)

// https://github.com/libyal/liblnk/blob/master/documentation/Windows%20Shortcut%20File%20(LNK)%20format.asciidoc#6-extra-data

func parseExtraBlocks(header *Header, offset int64, r io.ReaderAt) (*Extra, error) {
	var size uint32
	var signature uint32
	var data []byte
	var err error
	extra := &Extra{}
	for {
		size, signature, offset, data, err = readRawBlock(offset, r)
		if err != nil {
			return nil, err
		}
		if size == 0 {
			break
		}
		switch signature {
		case environmentBlock:
			extra.Environment, err = parseExtraEnvironment(size, data)
			if err != nil {
				return nil, err
			}
		case consoleBlock:
			extra.Console, err = parseExtraConsole(size, data)
			if err != nil {
				return nil, err
			}
		case trackerBlock:
			extra.Tracker, err = parseExtraTracker(size, data)
			if err != nil {
				return nil, err
			}
		case consoleFEBlock:
			extra.ConsoleFE, err = parseExtraConsoleFE(size, data)
			if err != nil {
				return nil, err
			}
		case specialFolderBlock:
			extra.SpecialFolder, err = parseExtraSpecialFolder(size, data)
			if err != nil {
				return nil, err
			}
		case darwinBlock:
			extra.Darwin, err = parseExtraDarwin(size, data)
			if err != nil {
				return nil, err
			}
		case iconEnvironmentBlock:
			extra.IconEnvironment, err = parseExtraIconEnvironment(size, data)
			if err != nil {
				return nil, err
			}
		case shimBlock:
			extra.Shim, err = parseExtraShim(size, data)
			if err != nil {
				return nil, err
			}
		case propertyStoreBlock:
			extra.PropertyStore, err = parseExtraPropertyStore(size, data)
			if err != nil {
				return nil, err
			}
		case knownFolderBlock:
			extra.KnownFolder, err = parseExtraKnownFolder(size, data)
			if err != nil {
				return nil, err
			}
		case vistaAndAboveIDListBlock:
			extra.VistaAndAboveIDList, err = parseExtraVistaAndAboveIDList(size, data)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown block signature: %x", signature)
		}
	}
	return extra, nil
}

func readRawBlock(offset int64, r io.ReaderAt) (uint32, uint32, int64, []byte, error) {
	size, data, err := readU32Data(offset, r)
	if err != nil {
		return 0, 0, 0, nil, err
	}
	if size == 0 {
		return 0, 0, 0, nil, nil
	}
	if size < 8 {
		return 0, 0, 0, nil, errors.New("invalid block size")
	}
	return size, binary.LittleEndian.Uint32(data[4:8]), offset + int64(size), data, nil
}
