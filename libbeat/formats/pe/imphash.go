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
	"crypto/md5"
	"debug/pe"
	"encoding/binary"
	"encoding/hex"
	"path/filepath"
	"strings"
)

func readString(section []byte, start int) string {
	if start < 0 || start >= len(section) {
		return ""
	}

	for end := start; end < len(section); end++ {
		if section[end] == 0 {
			return string(section[start:end])
		}
	}
	return ""
}

func importDirectory(f *pe.File) pe.DataDirectory {
	var emptyDirectory pe.DataDirectory
	if f.Machine == pe.IMAGE_FILE_MACHINE_AMD64 {
		header := f.OptionalHeader.(*pe.OptionalHeader64)
		if header.NumberOfRvaAndSizes < pe.IMAGE_DIRECTORY_ENTRY_IMPORT+1 {
			return emptyDirectory
		}
		return header.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_IMPORT]
	}
	header := f.OptionalHeader.(*pe.OptionalHeader32)
	if header.NumberOfRvaAndSizes < pe.IMAGE_DIRECTORY_ENTRY_IMPORT+1 {
		return emptyDirectory
	}
	return header.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_IMPORT]
}

func exportDirectory(f *pe.File) pe.DataDirectory {
	var emptyDirectory pe.DataDirectory
	if f.Machine == pe.IMAGE_FILE_MACHINE_AMD64 {
		header := f.OptionalHeader.(*pe.OptionalHeader64)
		if header.NumberOfRvaAndSizes < pe.IMAGE_DIRECTORY_ENTRY_EXPORT+1 {
			return emptyDirectory
		}
		return header.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT]
	}
	header := f.OptionalHeader.(*pe.OptionalHeader32)
	if header.NumberOfRvaAndSizes < pe.IMAGE_DIRECTORY_ENTRY_EXPORT+1 {
		return emptyDirectory
	}
	return header.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT]
}

func directoryData(f *pe.File, directory pe.DataDirectory) ([]byte, uint32, uint32, error) {
	if directory.Size == 0 {
		return nil, 0, 0, nil
	}
	var section *pe.Section
	for _, s := range f.Sections {
		if s.VirtualAddress <= directory.VirtualAddress && directory.VirtualAddress < s.VirtualAddress+s.VirtualSize {
			section = s
			break
		}
	}
	if section == nil {
		return nil, 0, 0, nil
	}

	data, err := section.Data()
	if err != nil {
		return nil, 0, 0, err
	}
	return data, directory.VirtualAddress, section.VirtualAddress, nil
}

func importData(f *pe.File) ([]byte, uint32, uint32, error) {
	return directoryData(f, importDirectory(f))
}

func exportData(f *pe.File) ([]byte, uint32, uint32, error) {
	return directoryData(f, exportDirectory(f))
}

func normalizeLibraryName(name string) string {
	name = strings.ToLower(name)
	extension := filepath.Ext(name)
	if extension == ".ocx" ||
		extension == ".sys" ||
		extension == ".dll" {
		return name[:len(name)-4]
	}
	return name
}

func exports(f *pe.File) []string {
	if f.OptionalHeader == nil {
		return nil
	}
	data, exportAddress, sectionAddress, err := exportData(f)
	if err != nil {
		// couldn't find the proper data directory, swallow the error
		return nil
	}
	if data == nil {
		return nil
	}
	exportOffset := exportAddress - sectionAddress
	if int(exportOffset) > len(data) {
		return nil
	}
	tableData := data[exportOffset:]
	if len(tableData) < 40 {
		return nil
	}
	exportCount := int(binary.LittleEndian.Uint32(tableData[24:30]))
	nameOffset := binary.LittleEndian.Uint32(tableData[32:36])
	if len(data) < int(nameOffset-sectionAddress)+1 {
		return nil
	}
	nameRVATable := data[nameOffset-sectionAddress:]
	// The pointers are 32 bits each and are relative to the image base
	if len(nameRVATable) < 4*exportCount {
		return nil
	}

	functions := make([]string, exportCount)
	for offset := 0; offset < exportCount; offset++ {
		start := offset * 4
		symbolOffset := binary.LittleEndian.Uint32(nameRVATable[start : start+4])
		functions[offset] = readString(data, int(symbolOffset-sectionAddress))
	}

	return functions
}

func imphash(f *pe.File) (map[string][]string, string) {
	if f.OptionalHeader == nil {
		return nil, ""
	}

	pe64 := f.Machine == pe.IMAGE_FILE_MACHINE_AMD64
	data, importAddress, sectionAddress, err := importData(f)
	if err != nil {
		// swallow error
		return nil, ""
	}
	if data == nil {
		return nil, ""
	}

	importOffset := importAddress - sectionAddress
	if int(importOffset) > len(data) {
		return nil, ""
	}
	tableData := data[importOffset:]
	offset := 0
	symbols := make(map[string][]string)
	imphashEntries := []string{}
	for len(tableData) >= offset+20 {
		directoryData := tableData[offset:]
		firstThunk := binary.LittleEndian.Uint32(directoryData[0:4])
		if firstThunk == 0 {
			// check to see if the image is not bound
			firstThunk = binary.LittleEndian.Uint32(directoryData[16:20])
			if firstThunk == 0 {
				break
			}
		}

		name := binary.LittleEndian.Uint32(directoryData[12:16])
		dllOffset := int(name - sectionAddress)
		dllName := readString(data, dllOffset)
		normalizedDllName := normalizeLibraryName(dllName)
		functionOffset := int(firstThunk - sectionAddress)
		offset += 20

		for len(data) > functionOffset {
			functionData := data[functionOffset:]
			if pe64 { // 64bit
				if len(functionData) < 8 {
					return nil, ""
				}
				functionAddress := binary.LittleEndian.Uint64(functionData[0:8])
				if functionAddress == 0 {
					break
				}
				if functionAddress&0x8000000000000000 > 0 { // is Ordinal
					normalizedFunctionName := strings.ToLower(lookupOrdinal(dllName, int(functionAddress&0xffffffff)))
					imphashEntries = append(imphashEntries, normalizedDllName+"."+normalizedFunctionName)
				} else {
					functionName := readString(data, int(uint32(functionAddress)-sectionAddress+2))
					symbols[dllName] = append(symbols[dllName], functionName)
					normalizedFunctionName := strings.ToLower(functionName)
					imphashEntries = append(imphashEntries, normalizedDllName+"."+normalizedFunctionName)
				}
				functionOffset += 8
			} else { // 32bit
				if len(functionData) < 4 {
					return nil, ""
				}
				functionAddress := binary.LittleEndian.Uint32(functionData[0:4])
				if functionAddress == 0 {
					break
				}
				if functionAddress&0x80000000 > 0 { // is Ordinal
					normalizedFunctionName := strings.ToLower(lookupOrdinal(dllName, int(functionAddress&0x0000ffff)))
					imphashEntries = append(imphashEntries, normalizedDllName+"."+normalizedFunctionName)
				} else {
					functionName := readString(data, int(functionAddress-sectionAddress+2))
					symbols[dllName] = append(symbols[dllName], functionName)
					normalizedFunctionName := strings.ToLower(functionName)
					imphashEntries = append(imphashEntries, normalizedDllName+"."+normalizedFunctionName)
				}
				functionOffset += 4
			}
		}
	}

	hash := md5.Sum([]byte(strings.Join(imphashEntries, ",")))
	return symbols, hex.EncodeToString(hash[:])
}
