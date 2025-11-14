// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package lnk

import (
	"encoding/hex"
	"fmt"
	"bytes"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/resources"
)

// https://www.forensic-cheatsheet.com/EN/Artifact/(EN)+Shellbag
type ShellItemType string

const (
	ShellItemTypeUnknown          ShellItemType = "Unknown"
	ShellItemTypeTerminator       ShellItemType = "Terminator"
	ShellItemTypeRootFolder       ShellItemType = "RootFolder"
	ShellItemTypeDelegateFolder   ShellItemType = "DelegateFolder"
	ShellItemTypeVolume           ShellItemType = "Volume"
	ShellItemTypeDirectory        ShellItemType = "Directory"
	ShellItemTypeFile             ShellItemType = "File"
	ShellItemTypeNetworkLocation  ShellItemType = "NetworkLocation"
	ShellItemTypeCompressedFolder ShellItemType = "CompressedFolder"
	ShellItemTypeControlPanel     ShellItemType = "ControlPanel"
	ShellItemTypeUri              ShellItemType = "Uri"
)

type RootFolderShellItem struct {
	Guid *resources.GUID
}

func ParseRootFolderShellItem(data []byte) (*RootFolderShellItem, error) {
	fmt.Printf("Root Folder Shell Item Data:\n %s\n", hex.Dump(data))
	extensionBlockSig := []byte{0xef, 0xbe}
	for i := 0; i < len(data) - len(extensionBlockSig); i++ {
		if bytes.Equal(data[i:i+len(extensionBlockSig)], extensionBlockSig) {
			fmt.Printf("Extension Block Signature found at index %d\n", i)
		}
	}

	return &RootFolderShellItem{Guid: nil}, nil
}

type ShellItem struct {
	Size  uint16
	Data  []byte
	Value any
}

func NewShellItem(size uint16, data []byte) *ShellItem {
	shellItem := &ShellItem{Size: size, Data: data}
	var err error
	switch shellItem.Type() {
	case ShellItemTypeRootFolder:
		shellItem.Value, err = ParseRootFolderShellItem(shellItem.Data)
		if err != nil {
			return nil
		}
	default:
		shellItem.Value = shellItem.Data
	}
	return shellItem
}

func (s *ShellItem) Type() ShellItemType {
	switch s.RawType() {
	case 0x00:
		return ShellItemTypeTerminator
	case 0x1F:
		return ShellItemTypeRootFolder
	case 0x14:
		return ShellItemTypeRootFolder
	case 0x20, 0x2F:
		return ShellItemTypeVolume
	case 0x31:
		return ShellItemTypeDirectory
	case 0x32:
		return ShellItemTypeFile
	case 0x40, 0x4F:
		return ShellItemTypeNetworkLocation
	case 0x52:
		return ShellItemTypeCompressedFolder
	case 0x61:
		return ShellItemTypeUri
	case 0x70:
		return ShellItemTypeControlPanel
	case 0x74:
		return ShellItemTypeDelegateFolder
	}
	return ShellItemTypeUnknown
}

func (s *ShellItem) RawType() byte {
	return s.Data[0]
}
