// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package shell_items

// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x1f.cs
// 0x1F: RootDirectory
// 
// The RootDirectory shell item is used to store the GUID of a known folder.
// It is a 16-bit unsigned integer that represents the length of the GUID.
// The GUID is stored in the data field as a 16-byte array.
// The GUID is encoded in Windows-1252.
//
// The data field is the GUID.
// The known folder is the name of the known folder looked up in the guidMappings.

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/resources"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

type RootDirectoryShellItem struct {
	Guid *resources.GUID
	KnownFolder string
	size uint16
	data []byte
}

func (s *RootDirectoryShellItem) String() string {
	return fmt.Sprintf("%s: guid: %s, known folder: %s", s.Type(), s.Guid.String(), s.KnownFolder)
}

func (s *RootDirectoryShellItem) Type() ShellItemType {
	return ShellItemTypeRootDirectory
}

func (s *RootDirectoryShellItem) RawType() byte {
	return s.data[0]
}

func NewRootDirectoryShellItem(size uint16, data []byte, log *logger.Logger) (ShellItem, error) {
	// Skip the first 2 bytes 1 byte for index, 1 byte unknown value
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x1f.cs#L369
	guid, err := resources.NewGUID(data[2:18])
	if err != nil {
		log.Errorf("Error creating GUID: %v\n", err)
		return &GenericShellItem{size: size, data: data}, err
	}
	knownFolder, ok := guid.LookupKnownFolder(); if !ok {
		log.Infof("Unknown known folder for GUID: %s\n", guid.String())
		knownFolder = ""
	}
	return &RootDirectoryShellItem{Guid: guid, KnownFolder: knownFolder, size: size, data: data}, nil
}

func NewShellBag1F(size uint16, data []byte, log *logger.Logger) (ShellItem, error) {
	if size != 0x14 {
		return nil, fmt.Errorf("ShellBag1F: size is not 0x14")
	}
	return NewRootDirectoryShellItem(size, data, log)
}
