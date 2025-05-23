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

//nolint:errorlint,godox // Bad linters!
package file_integrity

import (
	"errors"
	"fmt"
	"strings"

	"github.com/elastic/toutoumomoma"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// exeObjParser performs a number of analyses on executable objects
// such as PE, ELF and Mach-O file formats. It places results in file.*
// in the final document.
//
// The fields populated by exeObjParser are:
//
//	{elf,macho,pe}:
//	  sections:
//	    - name
//	      physical_size
//	      virtual_size
//	      entropy
//	      var_entropy
//	  import_hash
//	  imphash
//	  symhash
//	  imports
//	  imports_names_entropy
//	  imports_names_var_entropy
//	  go_import_hash
//	  go_imports
//	  go_imports_names_entropy
//	  go_imports_names_var_entropy
//	  go_stripped
type exeObjParser map[string]bool

func (fields exeObjParser) Parse(dst mapstr.M, path string) (err error) {
	if dst == nil {
		return errors.New("cannot use nil dst for file parser")
	}
	defer func() {
		switch r := recover().(type) {
		case nil:
		case error:
			// This will catch runtime.Error panics differentially.
			// These are the most likely panics during the analysis.
			err = fmt.Errorf("error panic during executable parser analysis of %s: %w", path, r)
		default:
			err = fmt.Errorf("panic during executable parser analysis of %s: %v", path, r)
		}
	}()

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

	var details mapstr.M
	d, err := dst.GetValue(typ)
	if err != nil {
		if err != mapstr.ErrKeyNotFound {
			return fmt.Errorf("invalid destination key: %q not found in map", typ)
		}
		details = make(mapstr.M)
		dst[typ] = details
	} else {
		switch d := d.(type) {
		case mapstr.M:
			details = d
		default:
			return fmt.Errorf("cannot write %s details to %T", typ, d)
		}
	}

	if all := wantFields(fields, "file."+typ+".sections"); all || wantFields(fields,
		"file."+typ+".sections.name",
		"file."+typ+".sections.virtual_size",
		"file."+typ+".sections.physical_size",
		"file."+typ+".sections.entropy",
		"file."+typ+".sections.var_entropy",
	) {
		sections, err := f.Sections()
		if err != nil {
			return err
		}
		var (
			name       *string
			size       *uint64
			fileSize   *uint64
			entropy    *float64
			varEntropy *float64

			wantName, wantSize, wantFileSize, wantEntropy, wantVariance bool
		)
		if !all {
			wantName = wantFields(fields, "file."+typ+".sections.name")
			wantSize = wantFields(fields, "file."+typ+".sections.virtual_size")
			wantFileSize = wantFields(fields, "file."+typ+".sections.physical_size")
			wantEntropy = wantFields(fields, "file."+typ+".sections.entropy")
			wantVariance = wantFields(fields, "file."+typ+".sections.var_entropy")
		}
		if len(sections) != 0 {
			// TODO: Replace this []section with a []mapstr.M if additional
			// section attributes are added from another parser.
			stats := make([]objSection, len(sections))
			for i, s := range sections {
				s := s
				if all {
					name = &s.Name
					size = &s.Size
					entropy = &s.Entropy
					varEntropy = &s.VarEntropy
				} else {
					if wantName {
						name = &s.Name
					}
					if wantSize {
						size = &s.Size
					}
					if wantFileSize {
						fileSize = &s.FileSize
					}
					if wantEntropy {
						entropy = &s.Entropy
					}
					if wantVariance {
						varEntropy = &s.VarEntropy
					}
				}
				stats[i] = objSection{
					Name:       name,
					Size:       size,
					FileSize:   fileSize,
					Entropy:    entropy,
					VarEntropy: varEntropy,
				}
			}
			details["sections"] = stats
		}
	}

	if wantFields(fields,
		"file.pe.imphash",
		"file.macho.symhash",
		"file."+typ+".import_hash",
		"file."+typ+".imports",
		"file."+typ+".imports_names_entropy",
		"file."+typ+".imports_names_var_entropy",
	) {
		h, symbols, err := f.ImportHash()
		if err != nil {
			return err
		}
		imphash := Digest(h)
		if wantFields(fields, "file."+typ+".import_hash") {
			details["import_hash"] = imphash
		}
		switch typ {
		case "pe":
			if wantFields(fields, "file.pe.imphash") {
				details["imphash"] = imphash
			}
		case "macho":
			if wantFields(fields, "file.macho.symhash") {
				details["symhash"] = imphash
			}
		}
		if len(symbols) != 0 {
			if wantFields(fields, "file."+typ+".imports") {
				details["imports"] = symbols
			}
			wantEntropy := wantFields(fields, "file."+typ+".imports_names_entropy")
			wantVariance := wantFields(fields, "file."+typ+".imports_names_var_entropy")
			if wantEntropy || wantVariance {
				entropy, varEntropy := toutoumomoma.NameEntropy(symbols)
				if wantEntropy {
					details["imports_names_entropy"] = entropy
				}
				if wantVariance {
					details["imports_names_var_entropy"] = varEntropy
				}
			}
		}
	}

	if wantFields(fields,
		"file."+typ+".go_import_hash",
		"file."+typ+".go_imports",
		"file."+typ+".go_imports_names_entropy",
		"file."+typ+".go_imports_names_var_entropy",
	) {
		h, symbols, err := f.GoSymbolHash(false)
		if err != nil {
			if err == toutoumomoma.ErrNotGoExecutable {
				return nil
			}
			return err
		}
		if wantFields(fields, "file."+typ+".go_import_hash") {
			details["go_import_hash"] = Digest(h)
		}
		if len(symbols) != 0 {
			if wantFields(fields, "file."+typ+".go_imports") {
				details["go_imports"] = symbols
			}
			wantEntropy := wantFields(fields, "file."+typ+".go_imports_names_entropy")
			wantVariance := wantFields(fields, "file."+typ+".go_imports_names_variance")
			if wantEntropy || wantVariance {
				entropy, varEntropy := toutoumomoma.NameEntropy(symbols)
				if wantEntropy {
					details["go_imports_names_entropy"] = entropy
				}
				if wantVariance {
					details["go_imports_names_var_entropy"] = varEntropy
				}
			}
		}
	}

	if wantFields(fields, "file."+typ+".go_stripped") {
		stripped, err := f.Stripped()
		if err != nil {
			return err
		}
		details["go_stripped"] = stripped
	}

	return nil
}

type objSection struct {
	Name       *string  `json:"name,omitempty"`
	Size       *uint64  `json:"virtual_size,omitempty"`
	FileSize   *uint64  `json:"physical_size,omitempty"`
	Entropy    *float64 `json:"entropy,omitempty"`
	VarEntropy *float64 `json:"var_entropy,omitempty"`
}
