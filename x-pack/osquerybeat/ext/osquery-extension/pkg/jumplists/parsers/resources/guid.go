// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows
// generate the guid_generated.go file
//go:generate go run ../../generate -output=guid_mapping_generated.go
package resources

import (
	"encoding/binary"
	"fmt"
	"strings"
)

type GuidType string
const (
	GuidTypeUnknown GuidType = "Unknown"
	GuidTypeFolder GuidType  = "Folder"
)

// GuidExtraData contains extra data about a GUID.
// Type is the type of the GUID.
// KnownFolder is the name of the known folder if the GUID is a known folder.
type GuidExtraData struct {
	Type GuidType
	KnownFolder string
}

// GUID represents a Windows GUID.
// Data1 is the first 4 bytes of the GUID.
// Data2 is the next 2 bytes of the GUID.
// Data3 is the next 2 bytes of the GUID.
// Data4 is the last 8 bytes of the GUID.
// ExtraData contains extra data about the GUID.
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
	ExtraData GuidExtraData
}

// String returns the string representation of the GUID.
func (g *GUID) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("%08X-%04X-%04X-", g.Data1, g.Data2, g.Data3))
	sb.WriteString(fmt.Sprintf("%02X%02X-", g.Data4[0], g.Data4[1]))
	sb.WriteString(fmt.Sprintf("%02X%02X%02X%02X%02X%02X", g.Data4[2], g.Data4[3], g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7]))
	return sb.String()
}

// NewGUID creates a new GUID from a 16-byte slice.
func NewGUID(data []byte) (*GUID, error) {
	if len(data) != 16 {
		return nil, fmt.Errorf("GUID must be 16 bytes long, got %d", len(data))
	}
	var data4 [8]byte
	copy(data4[:], data[8:16]) // Data4 is big-endian
	guid := &GUID{
		ExtraData: GuidExtraData{
			Type: GuidTypeUnknown,
			KnownFolder: "",
		},
		Data1: binary.LittleEndian.Uint32(data[:4]),
		Data2: binary.LittleEndian.Uint16(data[4:6]),
		Data3: binary.LittleEndian.Uint16(data[6:8]),
		Data4: data4,
	}

	if knownFolder, ok := LookupKnownFolder(guid.String()); ok {
		guid.ExtraData.Type = GuidTypeFolder
		guid.ExtraData.KnownFolder = knownFolder
	}

	return guid, nil
}

// LookupKnownFolder looks up a known folder by GUID.
func LookupKnownFolder(guid string) (string, bool) {
	if knownFolder, ok := knownFolderGuids[guid]; ok {
		return knownFolder, true
	}
	return "", false
}
