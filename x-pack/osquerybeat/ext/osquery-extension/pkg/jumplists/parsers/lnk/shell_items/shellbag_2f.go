// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package shell_items

import (
	"fmt"

	"golang.org/x/text/encoding/charmap"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x2f.cs
// 0x2F: DriveLetter
// 
// The DriveLetter shell item is used to store the drive letter of a drive.
// It is a 16-bit unsigned integer that represents the length of the drive letter.
// The drive letter is stored in the data field as a 2-byte array.
// The drive letter is encoded in Windows-1252.
//
// The data field is the drive letter.

type DriveLetterShellItem struct {
	DriveLetter string
	size uint16
	data []byte
}

func (s *DriveLetterShellItem) String() string {
	return fmt.Sprintf("%s: %s", s.Type(), s.DriveLetter)
}

func (s *DriveLetterShellItem) Type() ShellItemType {
	return ShellItemTypeDriveLetter
}

func (s *DriveLetterShellItem) RawType() byte {
	return s.data[0]
}

func (s *DriveLetterShellItem) Value() any {
	return s.data
}

func NewDriveLetterShellItem(size uint16, data []byte, log *logger.Logger) (ShellItem, error) {
	// drive letter starts at index 1 and is 2 bytes long
	driveLetter, err := charmap.Windows1252.NewDecoder().Bytes(data[1:3])
	if err != nil {
		 return nil, fmt.Errorf("Error getting encoding: %v\n", err)
	}
	return &DriveLetterShellItem{DriveLetter: string(driveLetter), size: size, data: data}, nil
}

func NewShellBag2F(size uint16, data []byte, log *logger.Logger) (ShellItem, error) {
	return NewDriveLetterShellItem(size, data, log)
}
