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

package file_integrity

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kortschak/toutoumomoma"

	"github.com/elastic/beats/v7/libbeat/common"
)

// exeObjParser performs a number of analyses on executable objects
// such as PE, ELF and Mach-O file formats. It places results in file.*
// in the final document.
//
// The fields populated by exeObjParser are:
//
//  {elf,macho,pe,plan9}:
//    sections:
//      - name
//        virtual_size
//        entropy
//    import_hash
//    imphash
//    symhash
//    imports
//    imports_names_entropy
//    go_import_hash
//    go_imports
//    go_imports_names_entropy
//    go_stripped
type exeObjParser map[string]bool

func (fields exeObjParser) Parse(dst common.MapStr, path string) error {
	if dst == nil {
		return errors.New("cannot use nil dst for file parser")
	}

	f, err := toutoumomoma.Open(path)
	if err != nil {
		if err == toutoumomoma.ErrUnknownFormat {
			return nil
		}
		return err
	}
	defer f.Close()

	typ := strings.ToLower(f.Type())
	if typ == "mach-o" {
		typ = "macho"
	}

	var details common.MapStr
	d, err := dst.GetValue(typ)
	if err != nil {
		if err != common.ErrKeyNotFound {
			panic(err)
		}
		details = make(common.MapStr)
		dst.Put(typ, details)
	} else {
		switch d := d.(type) {
		case common.MapStr:
			details = d
		default:
			return fmt.Errorf("cannot write %s details to %T", typ, d)
		}
	}

	if all := wantFields(fields, "file."+typ+".sections"); all || wantFields(fields,
		"file."+typ+".sections.name",
		"file."+typ+".sections.virtual_size",
		"file."+typ+".sections.entropy",
	) {
		sections, err := f.Sections()
		if err != nil {
			return err
		}
		var (
			name    *string
			size    *uint64
			entropy *float64

			wantName, wantSize, wantEntropy bool
		)
		if !all {
			wantName = wantFields(fields, "file."+typ+".sections.name")
			wantSize = wantFields(fields, "file."+typ+".sections.virtual_size")
			wantEntropy = wantFields(fields, "file."+typ+".sections.entropy")
		}
		if len(sections) != 0 {
			// TODO: Replace this []section with a []common.MapStr if additional
			// section attributes are added from another parser.
			stats := make([]objSection, len(sections))
			for i, s := range sections {
				s := s
				if all {
					name = &s.Name
					size = &s.Size
					entropy = &s.Entropy
				} else {
					if wantName {
						name = &s.Name
					}
					if wantSize {
						size = &s.Size
					}
					if wantEntropy {
						entropy = &s.Entropy
					}
				}
				stats[i] = objSection{
					Name:    name,
					Size:    size,
					Entropy: entropy,
				}
			}
			details.Put("sections", stats)
		}
	}

	if typ != "plan9" && // Plan9 is purely statically linked.
		wantFields(fields,
			"file.pe.imphash",
			"file.macho.symhash",
			"file."+typ+".import_hash",
			"file."+typ+".imports",
			"file."+typ+".imports_names_entropy",
		) {
		h, symbols, err := f.ImportHash()
		if err != nil {
			return err
		}
		imphash := Digest(h)
		if wantFields(fields, "file."+typ+".import_hash") {
			details.Put("import_hash", imphash)
		}
		switch typ {
		case "pe":
			if wantFields(fields, "file.pe.imphash") {
				details.Put("imphash", imphash)
			}
		case "macho":
			if wantFields(fields, "file.macho.symhash") {
				details.Put("symhash", imphash)
			}
		}
		if len(symbols) != 0 {
			if wantFields(fields, "file."+typ+".imports") {
				details.Put("imports", symbols)
			}
			if wantFields(fields, "file."+typ+".imports_names_entropy") {
				details.Put("imports_names_entropy", toutoumomoma.NameEntropy(symbols))
			}
		}
	}

	if wantFields(fields,
		"file."+typ+".go_import_hash",
		"file."+typ+".go_imports",
		"file."+typ+".go_imports_names_entropy",
	) {
		h, symbols, err := f.GoSymbolHash(false)
		if err != nil {
			if err == toutoumomoma.ErrNotGoExecutable {
				return nil
			}
			return err
		}
		if wantFields(fields, "file."+typ+".go_import_hash") {
			details.Put("go_import_hash", Digest(h))
		}
		if len(symbols) != 0 {
			if wantFields(fields, "file."+typ+".go_imports") {
				details.Put("go_imports", symbols)
			}
			if wantFields(fields, "file."+typ+".go_imports_names_entropy") {
				details.Put("go_imports_names_entropy", toutoumomoma.NameEntropy(symbols))
			}
		}
	}

	if wantFields(fields, "file."+typ+".go_stripped") {
		stripped, err := f.Stripped()
		if err != nil {
			return err
		}
		details.Put("go_stripped", stripped)
	}

	return nil
}

type objSection struct {
	Name    *string  `json:"name,omitempty"`
	Size    *uint64  `json:"virtual_size,omitempty"`
	Entropy *float64 `json:"entropy,omitempty"`
}
