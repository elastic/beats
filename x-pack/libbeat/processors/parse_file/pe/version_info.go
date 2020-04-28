// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pe

import (
	"bytes"
	"encoding/binary"
)

var (
	stringFileInfo = []byte{83, 0, 116, 0, 114, 0, 105, 0, 110, 0, 103, 0, 70, 0, 105, 0, 108, 0, 101, 0, 73, 0, 110, 0, 102, 0, 111, 0}
)

type versionInfo struct {
	Name  string
	Value string
}

func readStrings(data []byte) []versionInfo {
	childStrings := []versionInfo{}
	offset := 0
	for {
		if len(data) < offset+2 {
			return childStrings
		}
		stringData := data[offset:]
		stringSize := binary.LittleEndian.Uint16(stringData[0:2])
		if stringSize == 0 {
			offset += 2
			continue
		}
		valueType := binary.LittleEndian.Uint16(stringData[4:6])
		if valueType == 1 {
			key := readUnicode(stringData[6:])
			paddingOffset := len(key)*2 + 8
			paddedOffset := paddingOffset + (paddingOffset % 4)
			if len(stringData) >= paddedOffset+1 {
				value := readUnicode(stringData[paddedOffset:])
				if value != "" {
					childStrings = append(childStrings, versionInfo{
						Name:  key,
						Value: value,
					})
				}
			}
		}
		offset += int(stringSize)
	}
}

func readStringTables(data []byte) []versionInfo {
	childStrings := []versionInfo{}
	offset := 0
	for {
		if len(data) < offset+2 {
			return childStrings
		}
		tableData := data[offset:]
		tableSize := binary.LittleEndian.Uint16(tableData[0:2])
		if tableSize == 0 {
			offset += 2
			continue
		}
		// An 8-digit hexadecimal number stored as a Unicode string
		szKeyLength := 8 * 2
		childOffset := szKeyLength + 6
		paddedOffset := childOffset + (childOffset % 4)
		children := tableData[paddedOffset : int(tableSize)-paddedOffset]

		childStrings = append(childStrings, readStrings(children)...)
		offset += int(tableSize)
	}
}

func readStringFileInfo(data []byte) []versionInfo {
	szKeyLength := len(stringFileInfo)
	for i := 0; i < len(data)-szKeyLength; i++ {
		szKey := data[i : i+szKeyLength]
		if bytes.Compare(szKey, stringFileInfo) == 0 {
			return readStringTables(data[i+szKeyLength+(i+szKeyLength)%4:])
		}
	}
	return nil
}

func getVersionInfoForResources(resources []resource) ([]versionInfo, error) {
	for _, resource := range resources {
		if resource.Type == "RT_VERSION" {
			return readStringFileInfo(resource.data), nil
		}
	}
	return nil, nil
}
