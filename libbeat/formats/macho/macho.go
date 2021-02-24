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

package macho

import (
	"debug/macho"
	"fmt"
	"io"
	"sort"

	"github.com/elastic/beats/v7/libbeat/formats/common"
	"github.com/elastic/beats/v7/libbeat/formats/dwarf"
)

// Command contains info about a load command
type Command struct {
	Number int64  `json:"number"`
	Size   int64  `json:"size"`
	Type   string `json:"type,omitempty"`
}

// Header contains info about the overall file structure
type Header struct {
	Commands []Command `json:"commands"`
	Magic    string    `json:"magic"`
	Flags    []string  `json:"flags"`
}

// Section contains information about a section in a mach-o file.
type Section struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Offset    int64    `json:"offset"`
	Size      int64    `json:"size"`
	Entropy   float64  `json:"entropy"`
	ChiSquare float64  `json:"chi2"`
	Flags     []string `json:"flags,omitempty"`
}

// Segment contains info about a segment
type Segment struct {
	VMAddress  string    `json:"vmaddr"`
	Name       string    `json:"name"`
	VMSize     int64     `json:"vmsize"`
	FileOffset int64     `json:"fileoff"`
	FileSize   int64     `json:"filesize"`
	Sections   []Section `json:"sections,omitempty"`
	Flags      []string  `json:"flags,omitempty"`
}

// Architecture represents a fat file architecture
type Architecture struct {
	CPU       string        `json:"cpu"`
	ByteOrder string        `json:"byte_order"`
	Type      string        `json:"type,omitempty"`
	Header    Header        `json:"header"`
	Debug     []dwarf.DWARF `json:"debug,omitempty"`
	Segments  []Segment     `json:"segments,omitempty"`
	Libraries []string      `json:"libraries,omitempty"`
	Imports   []string      `json:"imports,omitempty"`
	Packers   []string      `json:"packers,omitempty"`
	Symhash   string        `json:"symhash,omitempty"`
	// TODO: Add the following
	// Exports   []string      `json:"exports,omitempty"`
	// CDHash    string        `json:"cdhash,omitempty"`
}

// Info contains high level fingerprinting an analysis of a mach-o file.
type Info struct {
	Architectures []*Architecture `json:"architectures,omitempty"`
}

// Parse parses the mach-o file and returns information about it or errors.
func Parse(r io.ReaderAt) (interface{}, error) {
	machoFiles := []*macho.File{}
	machoFatFile, err := macho.NewFatFile(r)
	if err != nil {
		if err != macho.ErrNotFat {
			return nil, err
		}
		machoFile, err := macho.NewFile(r)
		if err != nil {
			return nil, err
		}
		machoFiles = append(machoFiles, machoFile)
	} else {
		for _, arch := range machoFatFile.Arches {
			machoFiles = append(machoFiles, arch.File)
		}
	}

	architectures := make([]*Architecture, len(machoFiles))
	for i, machoFile := range machoFiles {
		arch, err := parse(machoFile)
		if err != nil {
			return nil, err
		}
		architectures[i] = arch
	}
	return &Info{
		Architectures: architectures,
	}, nil
}

func parse(machoFile *macho.File) (*Architecture, error) {
	symhash, err := symhash(machoFile)
	if err != nil {
		return nil, err
	}
	libraries, err := machoFile.ImportedLibraries()
	if err != nil {
		return nil, err
	}
	importSymbols, err := machoFile.ImportedSymbols()
	if err != nil {
		if _, ok := err.(*macho.FormatError); !ok {
			return nil, err
		}
	}

	segmentMap := make(map[string]Segment)
	for _, section := range machoFile.Sections {
		var entropy float64
		var chiSquare float64

		data, err := section.Data()
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
		} else {
			entropy = common.Entropy(data)
			chiSquare = common.ChiSquare(data)
		}
		segment, found := segmentMap[section.Seg]
		if !found {
			segment = Segment{
				Name: section.Seg,
			}
			mSegment := machoFile.Segment(section.Seg)
			if mSegment != nil {
				segment.VMAddress = fmt.Sprintf("0x%x", mSegment.Addr)
				segment.VMSize = int64(mSegment.Memsz)
				segment.FileOffset = int64(mSegment.Offset)
				segment.FileSize = int64(mSegment.Filesz)
			}
		}
		segment.Sections = append(segment.Sections, Section{
			Name:      section.Name,
			Size:      int64(section.Size),
			Offset:    int64(section.Offset),
			Entropy:   entropy,
			ChiSquare: chiSquare,
			Type:      sectionType(section.Flags),
			Flags:     sectionFlags(section.Flags),
		})
		segmentMap[section.Seg] = segment
	}
	segments := []Segment{}
	for _, segment := range segmentMap {
		segments = append(segments, segment)
	}
	sort.Slice(segments, func(i, j int) bool {
		return segments[i].FileOffset < segments[j].FileOffset
	})

	info := &Architecture{
		CPU:       translateCPU(machoFile.Cpu, machoFile.SubCpu),
		ByteOrder: translateByteOrder(machoFile.ByteOrder.String()),
		Type:      machoFile.Type.String(),
		Header: Header{
			Magic:    fmt.Sprintf("0x%x", machoFile.Magic),
			Flags:    headerFlags(machoFile.Flags),
			Commands: loadCommands(machoFile),
		},
		Symhash:   symhash,
		Libraries: libraries,
		Imports:   importSymbols,
		Segments:  segments,
		Packers:   getPackers(machoFile),
	}

	if debug, err := machoFile.DWARF(); err == nil {
		// just ignore the error if we can't get DWARF information
		debugSymbols, err := dwarf.Parse(debug)
		if err == nil {
			info.Debug = debugSymbols
		}
	}
	return info, nil
}

func translateByteOrder(order string) string {
	switch order {
	case "BigEndian":
		return "big-endian"
	case "LittleEndian":
		return "little-endian"
	default:
		return "unknown"
	}
}

func getPackers(machoFile *macho.File) []string {
	for _, section := range machoFile.Sections {
		if section.Name == "upxTEXT" {
			return []string{"upx"}
		}
	}
	return nil
}
