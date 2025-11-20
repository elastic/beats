// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package shell_items

import (
	"encoding/binary"
	"fmt"
	"encoding/hex"
	"time"
	"bytes"
	// "encoding/hex"
	// "fmt"
	"golang.org/x/text/encoding/unicode"

	// "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/resources"
	//"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)


type ExtensionBlock struct {
	signature byte
	size uint16
	data []byte
}

type Beef0004 struct {
	// Header Fields
	extensionSize uint16
	version       uint16
	signature     uint32

	// Metadata
	createdOn    *time.Time
	lastAccessOn *time.Time
	identifier   uint16
	mftInfo []byte

	// String Data
	longName      string
	localisedName string
}

func (b *Beef0004) String() string {
	return fmt.Sprintf("Beef0004: extension size: %d, version: %d, signature: %X, created on: %s, last accessed on: %s, identifier: %d", b.extensionSize, b.version, b.signature, b.createdOn, b.lastAccessOn, b.identifier)
}

func NewBeef0004(data []byte) (*Beef0004, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("data too short for Beef0004 header")
	}

	block := &Beef0004{}

	// 1. Parse Header
	block.extensionSize = binary.LittleEndian.Uint16(data[0:2])
	block.version = binary.LittleEndian.Uint16(data[2:4])
	block.signature = binary.LittleEndian.Uint32(data[4:8])

	// Validate Signature
	if block.signature != 0xBEEF0004 {
		return nil, fmt.Errorf("invalid signature: expected 0xBEEF0004, got 0x%X", block.signature)
	}

	createdOn, err := extractDateTimeOffsetFromBytes(data[8:12])
	if err != nil {
		return nil, err
	}
	block.createdOn = createdOn

	lastAccessed, err := extractDateTimeOffsetFromBytes(data[12:16])	
	if err != nil {
		return nil, err
	}
	block.lastAccessOn = lastAccessed

	block.identifier = binary.LittleEndian.Uint16(data[16:18])
	index := 18

	if block.version >= 7 {
		index += 2 // skip empty bytes
		block.mftInfo = data[index:index+8]
		index += 8 // skip mft info
		index += 8 // skip unknown

	}

	if block.version >= 3 {
		index += 2
	} 
	if block.version >= 9 {
		index += 4
	}
	if block.version >= 8 {
		index += 4
	}

	longstringSize := len(data) - index
	fmt.Printf("Index: %d, Longstring size: %d, Data length: %d\n", index, longstringSize, len(data))
    longstringBytes := data[index:len(data)]

	utf16Decoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()

	var stringArray []string
    if len(longstringBytes) < 2 {
		stringArray = append(stringArray, "")
	} else {
		longstringBytes = longstringBytes[:len(longstringBytes)-2]
		if bytes.Equal(longstringBytes[:3], []byte{0x00, 0x00, 0x00}) {
			stringArray = append(stringArray, "")
		} else {
			startIndex := 0
			for i := 0; i < len(longstringBytes); i += 2 {
				if longstringBytes[i] == 0 && longstringBytes[i+1] == 0 {
					chunk := longstringBytes[startIndex:i]
					str, err := utf16Decoder.String(string(chunk))
					if err != nil {
						return nil, err
					}
					stringArray = append(stringArray, str)
					startIndex = i + 2
				}
			}
			if startIndex < len(longstringBytes) {
				chunk := longstringBytes[startIndex:]
				str, err := utf16Decoder.String(string(chunk))
				if err != nil {
					return nil, err
				}
				stringArray = append(stringArray, str)
			}
		}
	}
	for _, str := range stringArray {
		fmt.Printf("String: %s\n", str)
	}

	fmt.Printf("Long string size: %d, Long string bytes size: %d\n", longstringSize, len(longstringBytes))
	fmt.Printf("Long string bytes: %s\n", hex.Dump(longstringBytes))

	return block, nil

	// // 2. Parse Version-Specific Data	
	// var offset int
	
	// if block.Version >= 0x03 {
	// 	if len(data) < 18 {
	// 		return nil, fmt.Errorf("data too short for Beef0004 v3 metadata")
	// 	}

	// 	// Parse Creation Time (Offsets 8-11)
	// 	// Uses the MS-DOS 4-byte format
		

	// 	// Parse Last Access Time (Offsets 12-15)
	// 	block.LastAccessOn = ExtractDateTimeOffsetFromBytes(data[12:16])

	// 	// Parse Identifier (Offsets 16-17)
	// 	block.Identifier = binary.LittleEndian.Uint16(data[16:18])

	// 	// Strings start at Offset 18
	// 	offset = 18
	// } else {
	// 	// Handle legacy versions if necessary (rare in modern forensics)
	// 	// Typically, just skips dates and starts strings earlier or at fixed offsets.
	// 	offset = 8 
	// }

	// // 3. Parse Long Name (UTF-16 LE)
	// // Read until null terminator (0x00 0x00)
	// longName, bytesRead := readUnicodeString(data[offset:])
	// block.LongName = longName
	// offset += bytesRead

	// // 4. Parse Localised Name (UTF-16 LE)
	// // Usually appears immediately after the Long Name
	// if offset < len(data) {
	// 	localisedName, _ := readUnicodeString(data[offset:])
	// 	block.LocalisedName = localisedName
	// }

	// return block, nil
}
