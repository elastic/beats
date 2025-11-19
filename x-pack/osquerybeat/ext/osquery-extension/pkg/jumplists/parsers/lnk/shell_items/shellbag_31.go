// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package shell_items

import (
	"encoding/hex"
	"fmt"
	"os"
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
	LastModified time.Time
	size uint16
	data []byte
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
	//https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x31.cs#L54-L59

	//1 byte for signature, 1 byte unknown value, 4 bytes for length (which is always 0 for directories)
	index := 2 // signature (0x31) + unknown byte
	index += 4 // length (which is always 0 for directories)

	lastModified, err := extractDateTimeOffsetFromBytes(data[index:index+4])
	if err != nil {
		return nil, err
	}

	outfile := "directoryItem_data.bin"
    _, err = os.Stat(outfile); if err != nil && os.IsNotExist(err) {
		os.WriteFile(outfile, data, 0644)
	}	
	
	index += 4 // last modified
	index += 4 // attributes

	beefSignature := []byte{0x04, 0x00, 0xef, 0xbe}

    beefPos := bytes.Index(data, beefSignature)
	
	extensionBlockStart := -1
	if beefPos != -1 {
		extensionBlockStart = beefPos - 4
	}

	strLen := -1
	if extensionBlockStart != -1 {
		strLen = extensionBlockStart - index
	}

	fmt.Printf("strLen: %d\n", strLen);
	fmt.Printf("extensionBlockStart: %d\n", extensionBlockStart);
	fmt.Printf("beefPos: %d\n", beefPos);

	copyLen := 0
	if strLen < 0 {
		copyLen = 2
		for (index + copyLen < len(data) && data[index + copyLen] != 0) {
			copyLen += 2;
		}
	} else if (data[2] == 0x35) || (data[2] == 0x36) {
		copyLen = strLen
	} else {
		for (data[index + copyLen] != 0) {
			copyLen += 1
		}
	}
	copyBytes := data[index:index+copyLen]
	fmt.Printf("copyBytes: %s\n", hex.Dump(copyBytes));
	fmt.Printf("copyLen: %d\n", copyLen);
	fmt.Printf("strlen: %d\n", beefPos - index);
	fmt.Printf("data[0x27]: %X\n", data[0x27]);
	fmt.Printf("data[0x28]: %X\n", data[0x27]);
	fmt.Printf("data[0x29]: %X\n", data[0x27]);
	fmt.Printf("data[0x24]: %X\n", data[0x27]);
	fmt.Printf("data[0x26]: %X\n", data[0x27]);
	fmt.Printf("data[0x28]: %X\n", data[0x27]);

	index += 2
	directory := string(data[index:beefPos-4])

	return &DirectoryShellItem{Directory: directory, LastModified: *lastModified, size: size, data: data}, nil
}

func NewShellBag31(size uint16, data []byte, log *logger.Logger) (ShellItem, error) {
	fmt.Printf("NewShellBag31\n");
	return NewDirectoryShellItem(size, data, log)
}
