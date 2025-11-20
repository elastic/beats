// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package shell_items

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	//	"golang.org/x/text/encoding/charmap"

	"bytes"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x31.cs
// 0x31: Directory
//
// The Directory shell item is used to store the path of a directory.
// It is a 16-bit unsigned integer that represents the length of the directory path.
// The directory path is stored in the data field as a null-terminated string.
// The directory path is encoded in Windows-1252.
//
// The data field is the directory path.

type DirectoryShellItem struct {
	Directory string
	ShortName string
	LastModified time.Time
	size uint16
	data []byte
	ExtensionBlocks []Beef0004
}

func (s *DirectoryShellItem) RawType() byte {
	return s.data[0]
}

func (s *DirectoryShellItem) String() string {
	return fmt.Sprintf("%s: directory: %s, last modified: %s", s.Type(), s.Directory, s.LastModified.Format(time.RFC3339))
}

func (s *DirectoryShellItem) Type() ShellItemType {
	return ShellItemTypeDirectory
}

func NewDirectoryShellItem(size uint16, data []byte, log *logger.Logger) (ShellItem, error) {
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x31.cs#L54-L59

	directoryShellItem := &DirectoryShellItem{size: size, data: data}

	// 1 byte for signature, 1 byte unknown value, 4 bytes for length (which is always 0 for directories)
	index := 2 // signature (0x31) + unknown byte
	index += 4 // length (which is always 0 for directories)

	// read the last modified time
	lastModifiedBytes := data[index:index+4]
	lastModified, err := extractDateTimeOffsetFromBytes(lastModifiedBytes)
	if err != nil {
		return nil, err
	}
	directoryShellItem.LastModified = *lastModified
	
	index += 4 // 4 bytes for last modified
	index += 2 // 2 bytes for attributes

	// search for the extension block
	beefSignature := []byte{0x04, 0x00, 0xef, 0xbe}
    beefPos := bytes.Index(data, beefSignature)

	// TODO: Move this to a function/struct and handle the strings properly
	extensionBlockStart := -1
	if beefPos != -1 {
		extensionBlockStart = beefPos - 4 // rewind 4 bytes to get the start of the extension block
		directoryShellItem.ShortName = string(data[index:extensionBlockStart])
		index = extensionBlockStart
	} else {
		// no extension block found, read until the NULL byte
		nulBytePos := bytes.IndexByte(data[index:], 0x00)
		if nulBytePos != -1 {
			directoryShellItem.ShortName = string(data[index:nulBytePos]) // read until the NULL byte
		}
		index = nulBytePos
	}

	if extensionBlockStart == -1 {
		return directoryShellItem, nil
	}

	for index < len(data) {
		fmt.Printf("Extension block size: %s\n", hex.Dump(data[index:index+2]))
		fmt.Printf("Extension block size little endian: %d\n", binary.LittleEndian.Uint16(data[index:index+2]))
		fmt.Printf("Index: %d\n", index)
		fmt.Printf("data length: %d\n", len(data))
		extensionBlockSize := int(binary.LittleEndian.Uint16(data[index:index+2]))
		fmt.Printf("index + extension block size: %d\n", index + extensionBlockSize)
		if len(data) < index + extensionBlockSize {
			err = fmt.Errorf("Extension block size is too large: %d", extensionBlockSize)
			return nil, err
		}
		extensionBlockData := data[index:index+extensionBlockSize]
		fmt.Printf("Extension block data: %s\n", hex.Dump(extensionBlockData))
		fmt.Printf("Extension block size: %d\n", extensionBlockSize)
		index += extensionBlockSize
		extensionBlock, err := NewBeef0004(extensionBlockData)
		if err != nil {
			return nil, err
		}
		directoryShellItem.ExtensionBlocks = append(directoryShellItem.ExtensionBlocks, *extensionBlock)
		fmt.Printf("Extension block: %s\n", extensionBlock.String())
	}

	return directoryShellItem, nil
}

func NewShellBag31(size uint16, data []byte, log *logger.Logger) (ShellItem, error) {
	fmt.Printf("NewShellBag31\n");
	return NewDirectoryShellItem(size, data, log)
}
