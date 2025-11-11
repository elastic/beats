// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package parsers

// https://www.forensic-cheatsheet.com/EN/Artifact/(EN)+Shellbag
type ShellItemType string

const (
	ShellItemTypeUnknown          ShellItemType = "Unknown"
	ShellItemTypeTerminator       ShellItemType = "Terminator"
	ShellItemTypeRootFolder       ShellItemType = "RootFolder"
	ShellItemTypeVolume           ShellItemType = "Volume"
	ShellItemTypeDirectory        ShellItemType = "Directory"
	ShellItemTypeFile             ShellItemType = "File"
	ShellItemTypeNetworkLocation  ShellItemType = "NetworkLocation"
	ShellItemTypeCompressedFolder ShellItemType = "CompressedFolder"
	ShellItemTypeControlPanel     ShellItemType = "ControlPanel"
	ShellItemTypeUri              ShellItemType = "Uri"
)

func (s *ShellItemType) String() string {
	return string(*s)
}

type ShellItem struct {
	Data []byte
}

func (s *ShellItem) Size() int {
	return len(s.Data)
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
	}
	return ShellItemTypeUnknown
}

func (s *ShellItem) RawType() byte {
	return s.Data[0]
}
