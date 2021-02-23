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

package elf

import (
	"bytes"
	"debug/elf"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/elastic/beats/v7/libbeat/formats/common"
	"github.com/elastic/beats/v7/libbeat/formats/dwarf"
)

// Segment represents a program segment
type Segment struct {
	Name     string   `json:"name"`
	Sections []string `json:"sections"`
}

// Symbol contains information about a symbol
type Symbol struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Header contains information inside the elf header.
type Header struct {
	Class      string `json:"class"`
	Data       string `json:"data"`
	Machine    string `json:"machine"`
	OSAbi      string `json:"os_abi"`
	Type       string `json:"type"`
	Version    string `json:"version"`
	AbiVersion string `json:"abi_version"`
	Entrypoint string `json:"entrypoint"`

	// Is this either Version or AbiVersion?
	// ObjectVersion string `json:"object_version"`
}

// Section contains information about a section in an elf file.
type Section struct {
	Flags          []string `json:"flags,omitempty"`
	Name           string   `json:"name"`
	PhysicalOffset int64    `json:"physical_offset"`
	Type           string   `json:"type"`
	PhysicalSize   int64    `json:"physical_size"`
	VirtualAddress int64    `json:"virtual_address"`
	VirtualSize    int64    `json:"virtual_size"`
	Entropy        float64  `json:"entropy"`
	ChiSquare      float64  `json:"chi2"`
}

// Info contains high level fingerprinting an analysis of a elf file.
type Info struct {
	Imports         []Symbol      `json:"imports,omitempty"`
	Exports         []Symbol      `json:"exports,omitempty"`
	Telfhash        string        `json:"telfhash,omitempty"`
	Segments        []Segment     `json:"segments,omitempty"`
	SharedLibraries []string      `json:"shared_libraries,omitempty"`
	Header          Header        `json:"header"`
	Sections        []Section     `json:"sections,omitempty"`
	Packers         []string      `json:"packers,omitempty"`
	Debug           []dwarf.DWARF `json:"debug,omitempty"`

	// This isn't in ELF
	// CreationDate    time.Time  `json:"creation_date"`

	// These are already contained in Header
	// Architecture string     `json:"architecture"`
	// ByteOrder    string     `json:"byte_order"`
	// CPUType      string     `json:"cpu_type"`
}

// Parse parses the elf file and returns information about it or errors.
func Parse(r io.ReaderAt) (interface{}, error) {
	elfFile, err := elf.NewFile(r)
	if err != nil {
		return nil, err
	}
	telfhash, err := telfhash(elfFile)
	if err != nil {
		return nil, err
	}
	dynamicSymbols, err := elfFile.DynamicSymbols()
	if err != nil {
		if err != elf.ErrNoSymbols {
			return nil, err
		}
	}
	exports := []Symbol{}
	imports := []Symbol{}
	librarySet := make(map[string]struct{})
	for _, symbol := range dynamicSymbols {
		binding := elf.ST_BIND(symbol.Info)
		if binding == elf.STB_GLOBAL && symbol.Section == elf.SHN_UNDEF {
			// symbol is imported
			library := symbol.Library
			if library != "" {
				librarySet[library] = struct{}{}
			}
			imports = append(imports, Symbol{
				Name: symbol.Name,
				Type: elf.ST_TYPE(symbol.Info).String(),
			})
		} else if elf.ST_VISIBILITY(symbol.Other) == elf.STV_DEFAULT {
			// if we have a weak or globally bound symbol, it's exported
			if binding == elf.STB_GLOBAL || binding == elf.STB_WEAK {
				exports = append(exports, Symbol{
					Name: symbol.Name,
					Type: elf.ST_TYPE(symbol.Info).String(),
				})
			}
		}
	}
	libraries := []string{}
	for library := range librarySet {
		libraries = append(libraries, library)
	}

	header := Header{
		Class:      translateClass(elfFile.Class),
		Data:       translateData(elfFile.Data),
		Machine:    translateMachine(elfFile.Machine),
		OSAbi:      translateOSABI(elfFile.OSABI),
		Type:       translateType(elfFile.Type),
		Version:    translateVersion(elfFile.Version),
		AbiVersion: fmt.Sprintf("%d", elfFile.ABIVersion),
		Entrypoint: fmt.Sprintf("%x", elfFile.Entry),
	}

	segments := make(map[*elf.Prog][]string)
	sections := []Section{}
	for _, section := range elfFile.Sections {
		var entropy float64
		var chiSquare float64

		name := section.Name
		if name == "" {
			if section.Size == 0 {
				continue
			}
			name = "UKNOWN"
		}
		for _, prog := range elfFile.Progs {
			if prog.Off <= section.Offset && prog.Off+prog.Memsz > section.Offset {
				// program segments can overlap, so don't break early
				segments[prog] = append(segments[prog], name)
			}
		}

		data, err := section.Data()
		if err == nil {
			entropy = common.Entropy(data)
			chiSquare = common.ChiSquare(data)
		}
		sections = append(sections, Section{
			Flags:          translateSectionFlags(section.Flags),
			Name:           name,
			PhysicalOffset: int64(section.Offset),
			Type:           translateSectionType(section.Type),
			PhysicalSize:   int64(section.FileSize),
			VirtualAddress: int64(section.Addr),
			VirtualSize:    int64(section.Size),
			Entropy:        entropy,
			ChiSquare:      chiSquare,
		})
	}
	translatedSegments := make([]Segment, len(elfFile.Progs))
	for i, prog := range elfFile.Progs {
		sections, ok := segments[prog]
		if !ok {
			sections = []string{}
		}
		translatedSegments[i] = Segment{
			Name:     translateProgType(prog.Type),
			Sections: sections,
		}
	}

	info := &Info{
		Imports:         imports,
		Exports:         exports,
		Telfhash:        telfhash,
		Segments:        translatedSegments,
		SharedLibraries: libraries,
		Header:          header,
		Sections:        sections,
		Packers:         getPackers(elfFile),
	}

	if debug, err := elfFile.DWARF(); err == nil {
		// just ignore the error if we can't get DWARF information
		debugSymbols, err := dwarf.Parse(debug)
		if err == nil {
			info.Debug = debugSymbols
		}
	}

	return info, nil
}

func getPackers(elfFile *elf.File) []string {
	// this is expensive, figure out a way of making it less so
	for _, prog := range elfFile.Progs {
		data, err := ioutil.ReadAll(prog.Open())
		if err == nil {
			if bytes.Contains(data, []byte("UPX!")) {
				return []string{"upx"}
			}
		}
	}
	return nil
}
