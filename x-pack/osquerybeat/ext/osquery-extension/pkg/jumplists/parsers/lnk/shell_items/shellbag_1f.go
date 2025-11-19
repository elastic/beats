// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package shell_items


import (
	"fmt"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/resources"
)

type ShellBag1F struct {
	size uint16
	data []byte
}

func (s *ShellBag1F) Type() ShellItemType {
	return ShellItemTypeRootFolder
}

func (s *ShellBag1F) RawType() byte {
	return s.data[0]
}

func (s *ShellBag1F) Value() any {
	return s.data
}

type RootFolderShellItem struct {
	Guid *resources.GUID
	size uint16
	data []byte
}

func (s *RootFolderShellItem) Type() ShellItemType {
	return ShellItemTypeRootFolder
}

func (s *RootFolderShellItem) RawType() byte {
	return s.data[0]
}

func (s *RootFolderShellItem) Value() any {
	return s.data
}
func NewRootFolderShellItem(size uint16, data []byte) ShellItem {
	// Skip the first 2 bytes 1 byte for index, 1 byte unknown value
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x1f.cs#L369
	guid, err := resources.NewGUID(data[2:18])
	if err != nil {
		return &GenericShellItem{size: size, data: data}
	}

	fmt.Printf("GUID: %s\n", guid.String())
	return &RootFolderShellItem{Guid: guid, size: size, data: data}
}

func NewShellBag1F(size uint16, data []byte) ShellItem {
	fmt.Printf("Size: %X\n", size)
	if size == 0x14 {
		return NewRootFolderShellItem(size, data)
	}
	
	if data[0] != 0x1F {
		return nil
	}
	off3Bitmask := data[1] & 0x70
	fmt.Printf("off3Bitmask: %X\n", off3Bitmask)
	if data[4] == 0x2F {
		fmt.Printf("GUID ONLY")
	}
	return &ShellBag1F{size: size, data: data}
}