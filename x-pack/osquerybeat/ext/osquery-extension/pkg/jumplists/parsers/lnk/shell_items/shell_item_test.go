// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package shell_items

import (
	"bytes"
	"fmt"
	"encoding/hex"
	"os"
	"testing"

	golnk "github.com/parsiya/golnk"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func TestShellItem(t *testing.T) {
	log := logger.New(os.Stdout, true)
	destinationFilePath := "../../../testdata/lnks/lnk_36.bin"
	fileBytes, err := os.ReadFile(destinationFilePath)
	if err != nil {
		t.Errorf("os.ReadFile() returned error: %v", err)
	}
	lnk, err := golnk.Read(bytes.NewReader(fileBytes), uint64(len(fileBytes)))
	if err != nil {
		t.Errorf("golnk.Read() returned error: %v", err)
	}

	if lnk.Header.LinkFlags["HasLinkTargetIDList"] {
		for _, item := range lnk.IDList.List.ItemIDList {
			fmt.Printf("item: %s\n", hex.Dump(item.Data))
			fmt.Printf("item size: %d\n", item.Size)
			fmt.Printf("item data size: %d\n", len(item.Data))
			shellItem := NewShellItem(item.Size, item.Data, log)
			if _, ok := shellItem.(*DirectoryShellItem); ok {
				return
			}
		}
	}


}

// func TestShellItem(t *testing.T) {
// 	shellItemBytes := []byte{0x1F, 0x50, 0xE0, 0x4F, 0xD0, 0x20, 0xEA, 0x3A, 0x69, 0x10, 0xA2, 0xD8, 0x08, 0x00, 0x2B, 0x30, 0x30, 0x9D}
// 	size := uint16(len(shellItemBytes)) + 2
// 	shellItem := NewShellItem(size, shellItemBytes)
// 	fmt.Printf("shellItem: %v\n", shellItem)
// 	si, ok := shellItem.(*RootFolderShellItem)
// 	if !ok {
// 		t.Fatalf("shellItem is not a RootFolderShellItem")
// 	}
// 	fmt.Printf("RootFolderShellItem: %v\n", si.Guid.String())
// }