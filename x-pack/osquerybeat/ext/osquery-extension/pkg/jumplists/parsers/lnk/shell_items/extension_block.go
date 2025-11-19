// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package shell_items

import (
	"fmt"
	// "bytes"
	// "encoding/hex"
	// "fmt"

	// "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/resources"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)


type ExtensionBlock struct {
	signature byte
	size uint16
	data []byte
}


func NewExtensionBlock(size uint16, data []byte, log *logger.Logger) ShellItem {
	signature := data[0]
	var shellItem ShellItem
	var err error

	switch signature {
	case 0x1F:
		shellItem, err = NewShellBag1F(size, data, log)
	case 0x2F:
		shellItem, err = NewShellBag2F(size, data, log)
	default:
		err = fmt.Errorf("Unknown shell item signature: %X\n", signature)
	}

	if err != nil {
		log.Errorf("Error creating shell item: %v\n", err)
		return &GenericShellItem{size: size, data: data}
	}

	return shellItem
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
