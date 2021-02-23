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

package dwarf

import (
	"debug/dwarf"
	"time"
)

var dwarfTypes = map[dwarf.Tag]string{
	dwarf.TagArrayType:              "array",
	dwarf.TagClassType:              "class",
	dwarf.TagEntryPoint:             "entrypoint",
	dwarf.TagEnumerationType:        "enumeration",
	dwarf.TagFormalParameter:        "formal parameter",
	dwarf.TagImportedDeclaration:    "imported declaration",
	dwarf.TagLabel:                  "label",
	dwarf.TagLexDwarfBlock:          "lex block",
	dwarf.TagMember:                 "member",
	dwarf.TagPointerType:            "pointer",
	dwarf.TagReferenceType:          "reference",
	dwarf.TagCompileUnit:            "compile unit",
	dwarf.TagStringType:             "string",
	dwarf.TagStructType:             "struct",
	dwarf.TagSubroutineType:         "subroutine",
	dwarf.TagTypedef:                "typedef",
	dwarf.TagUnionType:              "union",
	dwarf.TagUnspecifiedParameters:  "unspecified parameters",
	dwarf.TagVariant:                "variant",
	dwarf.TagCommonDwarfBlock:       "common block",
	dwarf.TagCommonInclusion:        "common inclusion",
	dwarf.TagInheritance:            "inheritance",
	dwarf.TagInlinedSubroutine:      "inlined subroutine",
	dwarf.TagModule:                 "module",
	dwarf.TagPtrToMemberType:        "pointer to member",
	dwarf.TagSetType:                "set",
	dwarf.TagSubrangeType:           "subrange",
	dwarf.TagWithStmt:               "with statement",
	dwarf.TagAccessDeclaration:      "access declaration",
	dwarf.TagBaseType:               "base",
	dwarf.TagCatchDwarfBlock:        "catch block",
	dwarf.TagConstType:              "const",
	dwarf.TagConstant:               "constant",
	dwarf.TagEnumerator:             "enumerator",
	dwarf.TagFileType:               "file",
	dwarf.TagFriend:                 "friend",
	dwarf.TagNamelist:               "namelist",
	dwarf.TagNamelistItem:           "namelist item",
	dwarf.TagPackedType:             "packed",
	dwarf.TagSubprogram:             "subprogram",
	dwarf.TagTemplateTypeParameter:  "template type parameter",
	dwarf.TagTemplateValueParameter: "template value parameter",
	dwarf.TagThrownType:             "thrown",
	dwarf.TagTryDwarfBlock:          "try block",
	dwarf.TagVariantPart:            "variant part",
	dwarf.TagVariable:               "variable",
	dwarf.TagVolatileType:           "volatile",
	dwarf.TagDwarfProcedure:         "procedure",
	dwarf.TagRestrictType:           "restrict",
	dwarf.TagInterfaceType:          "interface",
	dwarf.TagNamespace:              "namespace",
	dwarf.TagImportedModule:         "imported module",
	dwarf.TagUnspecifiedType:        "unspecified",
	dwarf.TagPartialUnit:            "partial unit",
	dwarf.TagImportedUnit:           "imported unit",
	dwarf.TagMutableType:            "mutable",
	dwarf.TagCondition:              "condition",
	dwarf.TagSharedType:             "shared",
	dwarf.TagTypeUnit:               "type unit",
	dwarf.TagRvalueReferenceType:    "rvalue reference",
	dwarf.TagTemplateAlias:          "template alias",
	dwarf.TagCoarrayType:            "coarray",
	dwarf.TagGenericSubrange:        "generic subrange",
	dwarf.TagDynamicType:            "dynamic",
	dwarf.TagAtomicType:             "atomic",
	dwarf.TagCallSite:               "call site",
	dwarf.TagCallSiteParameter:      "call site parameter",
	dwarf.TagSkeletonUnit:           "skeleton unit",
	dwarf.TagImmutableType:          "immutable",
}

func lookupType(tag dwarf.Tag) string {
	if name, ok := dwarfTypes[tag]; ok {
		return name
	}
	return "unknown"
}

// DWARF contains debug info
type DWARF struct {
	Offset    int64      `json:"offset,omitempty"`
	Size      int64      `json:"size,omitempty"`
	Type      string     `json:"type,omitempty"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
}

// Parse parses a DWARF table into debug sections
func Parse(data *dwarf.Data) ([]DWARF, error) {
	reader := data.Reader()
	if reader == nil {
		return nil, nil
	}
	offset := dwarf.Offset(0)
	symbols := []DWARF{}
	for {
		entry, err := reader.Next()
		if entry == nil {
			break
		}
		if err != nil {
			return nil, err
		}
		size := entry.Offset - offset
		offset = entry.Offset
		var compiledAt *time.Time
		if entry.Tag == dwarf.TagCompileUnit {
			lreader, err := data.LineReader(entry)
			if err == nil && lreader != nil {
				// just skip if we can't read the data
				for _, f := range lreader.Files() {
					if f != nil && f.Mtime != 0 {
						// we have some sort of modification time
						// use it as thhe compiled time
						compiled := time.Unix(int64(f.Mtime), 0).UTC()
						compiledAt = &compiled
						break
					}
				}
			}
		}
		symbols = append(symbols, DWARF{
			Offset:    int64(entry.Offset),
			Size:      int64(size),
			Type:      lookupType(entry.Tag),
			Timestamp: compiledAt,
		})
	}
	return symbols, nil
}
