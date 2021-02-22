package elf

import (
	"bytes"
	"crypto/md5"
	"debug/elf"
	"encoding/hex"
	"io"
	"io/ioutil"

	"github.com/elastic/beats/v7/libbeat/formats/common"
)

// Section contains information about a section in a mach-o file.
type Section struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Address   uint64  `json:"address"`
	Size      uint64  `json:"size"`
	Offset    uint64  `json:"offset"`
	Entropy   float64 `json:"entropy"`
	ChiSquare float64 `json:"chi2"`
	Flags     string  `json:"flags"`
	MD5       string  `json:"md5,omitempty"`
}

// Segment represents a program segment
type Segment struct {
	Name     string   `json:"name"`
	Sections []string `json:"sections"`
}

// Info contains high level fingerprinting an analysis of a mach-o file.
type Info struct {
	Machine  string              `json:"machine"`
	Segments []Segment           `json:"segments,omitempty"`
	Sections []Section           `json:"sections,omitempty"`
	Imports  map[string][]string `json:"imports,omitempty"`
	Exports  []string            `json:"exports,omitempty"`
	Packer   string              `json:"packer,omitempty"`
	Telfhash string              `json:"telfhash,omitempty"`
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
	groupedSymbols := make(map[string][]string)
	importSymbols, err := elfFile.ImportedSymbols()
	if err != nil {
		if err != elf.ErrNoSymbols {
			return nil, err
		}
	}
	for _, symbol := range importSymbols {
		library := symbol.Library
		if library == "" {
			library = "unknown"
		}
		groupedSymbols[library] = append(groupedSymbols[library], symbol.Name)
	}

	segments := make(map[*elf.Prog][]string)
	sections := []Section{}
	for _, section := range elfFile.Sections {
		var md5String string
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
			md5hash := md5.Sum(data)
			md5String = hex.EncodeToString(md5hash[:])
			entropy = common.Entropy(data)
			chiSquare = common.ChiSquare(data)
		}
		sections = append(sections, Section{
			Name:      name,
			Type:      translateSectionType(section.Type),
			Address:   section.Addr,
			Size:      section.Size,
			Offset:    section.Offset,
			Entropy:   entropy,
			ChiSquare: chiSquare,
			Flags:     translateSectionFlags(section.Flags),
			MD5:       md5String,
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

	return &Info{
		Machine:  translateMachine(elfFile.Machine),
		Sections: sections,
		Segments: translatedSegments,
		Imports:  groupedSymbols,
		Packer:   getPacker(elfFile),
		Telfhash: telfhash,
	}, nil
}

func getPacker(elfFile *elf.File) string {
	// this is expensive, figure out a way of making it less so
	for _, prog := range elfFile.Progs {
		data, err := ioutil.ReadAll(prog.Open())
		if err == nil {
			if bytes.Contains(data, []byte("UPX!")) {
				return "upx"
			}
		}
	}
	return ""
}
