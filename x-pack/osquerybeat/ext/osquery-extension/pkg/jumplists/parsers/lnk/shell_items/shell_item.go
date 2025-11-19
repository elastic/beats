// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package shell_items

import (
	// "bytes"
	// "encoding/hex"
	// "fmt"

	// "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/resources"
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

type ShellItem interface {
	Type() ShellItemType
	RawType() byte
	Value() any
}

type GenericShellItem struct {
	size  uint16
	data  []byte
	value any
}

func (s *GenericShellItem) Type() ShellItemType {
	return ShellItemTypeUnknown
}

func (s *GenericShellItem) RawType() byte {
	return s.data[0]
}

func (s *GenericShellItem) Value() any {
	return s.value
}
func NewShellItem(size uint16, data []byte) ShellItem {
	signature := data[0]
	switch signature {
	case 0x1F:
		return NewShellBag1F(size, data)
	default:
		return &GenericShellItem{size: size, data: data}
	}
}

// func (s *GenericShellItem) Type() ShellItemType {
// 	switch s.RawType() {
// 	case 0x00:
// 		return ShellItemTypeTerminator
// 	case 0x1F:
// 		return ShellItemTypeRootFolder
// 	case 0x14:
// 		return ShellItemTypeRootFolder
// 	case 0x20, 0x2F:
// 		return ShellItemTypeVolume
// 	case 0x31:
// 		return ShellItemTypeDirectory
// 	case 0x32:
// 		return ShellItemTypeFile
// 	case 0x40, 0x4F:
// 		return ShellItemTypeNetworkLocation
// 	case 0x52:
// 		return ShellItemTypeCompressedFolder
// 	case 0x61:
// 		return ShellItemTypeUri
// 	case 0x70:
// 		return ShellItemTypeControlPanel
// 	case 0x74:
// 		return ShellItemTypeDelegateFolder
// 	}
// 	return ShellItemTypeUnknown
// }

// func (s *ShellItem) RawType() byte {
// 	return s.Data[0]
// }
