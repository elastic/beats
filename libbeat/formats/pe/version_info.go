// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package pe

import (
	"bytes"
	"encoding/binary"

	"github.com/elastic/beats/v7/libbeat/formats/common"
)

var (
	stringFileInfo = []byte{83, 0, 116, 0, 114, 0, 105, 0, 110, 0, 103, 0, 70, 0, 105, 0, 108, 0, 101, 0, 73, 0, 110, 0, 102, 0, 111, 0}
)

func readStrings(data []byte) []VersionInfo {
	childStrings := []VersionInfo{}
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
		if len(stringData) < 6 {
			// we have junk string, just try and read past the offset
			offset += int(stringSize)
			continue
		}
		valueType := binary.LittleEndian.Uint16(stringData[4:6])
		if valueType == 1 {
			key := common.ReadUnicode(stringData, 6)
			paddingOffset := len(key)*2 + 8
			paddedOffset := paddingOffset + (paddingOffset % 4)
			if len(stringData) >= paddedOffset+1 {
				value := common.ReadUnicode(stringData, paddedOffset)
				if value != "" {
					childStrings = append(childStrings, VersionInfo{
						Name:  key,
						Value: value,
					})
				}
			}
		}
		offset += int(stringSize)
	}
}

func readStringTables(data []byte) []VersionInfo {
	childStrings := []VersionInfo{}
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
		childEnd := int(tableSize) - paddedOffset
		if childEnd < paddedOffset || len(tableData) < paddedOffset+1 || len(tableData) < int(tableSize)-paddedOffset {
			// we have an invalid string
			offset += int(tableSize)
			continue
		}
		children := tableData[paddedOffset:childEnd]

		childStrings = append(childStrings, readStrings(children)...)
		offset += int(tableSize)
	}
}

func readStringFileInfo(data []byte) []VersionInfo {
	szKeyLength := len(stringFileInfo)
	if len(data) < szKeyLength {
		return nil
	}
	for i := 0; i < len(data)-szKeyLength; i++ {
		szKey := data[i : i+szKeyLength]
		if bytes.Compare(szKey, stringFileInfo) == 0 {
			return readStringTables(data[i+szKeyLength+(i+szKeyLength)%4:])
		}
	}
	return nil
}

func getVersionInfoForResources(resources []Resource) map[string]string {
	for _, resource := range resources {
		if resource.Type == "RT_VERSION" {
			versionInfo := readStringFileInfo(resource.data)
			data := make(map[string]string, len(versionInfo))
			for _, info := range versionInfo {
				data[info.Name] = info.Value
			}
			return data
		}
	}
	return nil
}
