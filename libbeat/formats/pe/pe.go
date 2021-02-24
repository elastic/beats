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
	"debug/pe"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/elastic/beats/v7/libbeat/formats/common"
	"github.com/elastic/beats/v7/libbeat/formats/dwarf"
)

// Header contains information found in a PE header.
type Header struct {
	CompilationTimestamp *time.Time `json:"compilationTimestamp,omitempty"`
	Entrypoint           uint32     `json:"entrypoint"`
	TargetMachine        string     `json:"targetMachine"`
	ContainedSections    int        `json:"containedSections"`
}

// VersionInfo hold keys and values parsed from the version info resource.
type VersionInfo struct {
	Name  string
	Value string
}

// Compiler contains compiler information about the object file
type Compiler struct {
	Version string `json:"version,omitempty"`
	Name    string `json:"name,omitempty"`
}

// ImportedSymbol contains information about where an imported symbol comes from
type ImportedSymbol struct {
	Library string `json:"library,omitempty"`
	Name    string `json:"name,omitempty"`
}

// Section contains information about a section in a PE file.
type Section struct {
	Name           string   `json:"name"`
	Flags          []string `json:"flags"`
	VirtualAddress uint32   `json:"virtual_address"`
	RawSize        uint32   `json:"raw_size,omitempty"`
	Entropy        float64  `json:"entropy,omitempty"`
	ChiSquare      float64  `json:"chi2,omitempty"`
}

// Resource represents a resource entry embedded in a PE file.
type Resource struct {
	Type      string  `json:"type"`
	Language  string  `json:"language"`
	SHA256    string  `json:"sha256"`
	FileType  string  `json:"filetype,omitempty"`
	Entropy   float64 `json:"entropy"`
	ChiSquare float64 `json:"chi2"`

	data []byte
}

// Icon holds fields that are used for fingerprinting embedded icons
type Icon struct {
	// leverage https://github.com/corona10/goimagehash
	Dhash string `json:"dhash"`
}

// Info contains high level fingerprinting an analysis of a PE file.
type Info struct {
	CompilationTimestamp *time.Time       `json:"compile_timestamp,omitempty"`
	Entrypoint           string           `json:"entrypoint"`
	Exports              []string         `json:"exports,omitempty"`
	Debug                []dwarf.DWARF    `json:"debug,omitempty"`
	Imports              []ImportedSymbol `json:"imports,omitempty"`
	Sections             []Section        `json:"sections,omitempty"`
	Resources            []Resource       `json:"resources,omitempty"`
	Packers              []string         `json:"packers,omitempty"`
	ImpHash              string           `json:"imphash,omitempty"`
	FileVersion          string           `json:"file_version,omitempty"`
	Description          string           `json:"description,omitempty"`
	Company              string           `json:"company,omitempty"`
	OriginalFileName     string           `json:"original_file_name,omitempty"`
	Product              string           `json:"product,omitempty"`
	Architecture         string           `json:"architecture,omitempty"`

	// Things that we should be able to get
	// See https://github.com/lief-project/LIEF/blob/05103f55a6cb993cb20735da3c7a6333e4f600e3/src/PE/Binary.cpp#L1046
	// Authentihash         string           `json:"authentihash,omitempty"`
	// Compiler         *Compiler        `json:"compiler,omitempty"`
	// RichHeaderHash   string           `json:"rich_header.hash.md5,omitempty"`
	// Icons            []Icon           `json:"icon,omitempty"`

	// Fields that are likely duplicated
	// CreationDate         *time.Time       `json:"creation_date,omitempty"`
	// MachineType          string           `json:"machine_type"`
}

func getPackers(f *pe.File) []string {
	for _, section := range f.Sections {
		if section.Name == "UPX0" {
			return []string{"upx"}
		}
	}
	return nil
}

// Parse parses the PE and returns information about it or errors.
func Parse(r io.ReaderAt) (interface{}, error) {
	peFile, err := pe.NewFile(r)
	if err != nil {
		return nil, err
	}
	// IsDLL:        (peFile.Characteristics & 0x2000) == 0x2000,
	// IsSys:        (peFile.Characteristics & 0x1000) == 0x1000,

	var architecture string
	var entrypoint uint32
	switch header := peFile.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		architecture = "x32"
		entrypoint = header.AddressOfEntryPoint

	case *pe.OptionalHeader64:
		architecture = "x64"
		entrypoint = header.AddressOfEntryPoint

	default:
		architecture = "unknown"
	}

	exportSymbols := exports(peFile)
	importSymbols, imphash := imphash(peFile)
	imports := []ImportedSymbol{}
	for library, symbols := range importSymbols {
		for _, symbol := range symbols {
			imports = append(imports, ImportedSymbol{
				Library: library,
				Name:    symbol,
			})
		}
	}
	sort.Slice(imports, func(i, j int) bool {
		return (imports[i].Library < imports[j].Library && imports[i].Name < imports[j].Name)
	})

	sectionSize := len(peFile.Sections)
	var compiledAt *time.Time
	timestamp := int64(peFile.FileHeader.TimeDateStamp)
	if timestamp != 0 {
		compiled := time.Unix(timestamp, 0).UTC()
		compiledAt = &compiled
	}

	info := &Info{
		CompilationTimestamp: compiledAt,
		Entrypoint:           fmt.Sprintf("%x", entrypoint),
		Imports:              imports,
		Exports:              exportSymbols,
		Packers:              getPackers(peFile),
		ImpHash:              imphash,
		Architecture:         architecture,
		Sections:             make([]Section, sectionSize),
	}

	if debug, err := peFile.DWARF(); err == nil {
		// just ignore the error if we can't get DWARF information
		debugSymbols, err := dwarf.Parse(debug)
		if err == nil {
			info.Debug = debugSymbols
		}
	}

	for i, section := range peFile.Sections {
		data, _ := section.Data()
		info.Sections[i] = Section{
			Name:           section.Name,
			VirtualAddress: section.VirtualAddress,
			RawSize:        section.Size,
			Flags:          translateSectionFlags(section.Characteristics),
			Entropy:        common.Entropy(data),
			ChiSquare:      common.ChiSquare(data),
		}

		if section.Name == ".rsrc" && len(data) > 0 {
			info.Resources = parseDirectory(section.VirtualAddress, data)
			fileVersionInfo := getVersionInfoForResources(info.Resources)
			if companyName, found := fileVersionInfo["CompanyName"]; found {
				info.Company = companyName
			}
			if fileDescription, found := fileVersionInfo["FileDescription"]; found {
				info.Description = fileDescription
			}
			if fileVersion, found := fileVersionInfo["FileVersion"]; found {
				info.FileVersion = fileVersion
			}
			if originalFilename, found := fileVersionInfo["OriginalFilename"]; found {
				info.OriginalFileName = originalFilename
			}
			productName := fileVersionInfo["ProductName"]
			if productVersion, found := fileVersionInfo["ProductVersion"]; productName != "" && found {
				productName += " (" + productVersion + ")"
			}
			if productName != "" {
				info.Product = productName
			}
		}
	}
	return info, nil
}
