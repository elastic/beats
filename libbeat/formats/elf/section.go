package elf

import (
	"debug/elf"
	"strings"
)

var sectionNames = map[elf.SectionType]string{
	elf.SHT_NULL:           "NULL",
	elf.SHT_PROGBITS:       "PROGBITS",
	elf.SHT_SYMTAB:         "SYMTAB",
	elf.SHT_STRTAB:         "STRTAB",
	elf.SHT_RELA:           "RELA",
	elf.SHT_HASH:           "HASH",
	elf.SHT_DYNAMIC:        "DYNAMIC",
	elf.SHT_NOTE:           "NOTE",
	elf.SHT_NOBITS:         "NOBITS",
	elf.SHT_REL:            "REL",
	elf.SHT_SHLIB:          "SHLIB",
	elf.SHT_DYNSYM:         "DYNSYM",
	elf.SHT_INIT_ARRAY:     "INIT_ARRAY",
	elf.SHT_FINI_ARRAY:     "FINI_ARRAY",
	elf.SHT_PREINIT_ARRAY:  "PREINIT_ARRAY",
	elf.SHT_GROUP:          "GROUP",
	elf.SHT_SYMTAB_SHNDX:   "SYMTAB_SHNDX",
	elf.SHT_GNU_ATTRIBUTES: "GNU_ATTRIBUTES",
	elf.SHT_GNU_HASH:       "GNU_HASH",
	elf.SHT_GNU_LIBLIST:    "GNU_LIBLIST",
	elf.SHT_GNU_VERDEF:     "GNU_VERDEF",
	elf.SHT_GNU_VERNEED:    "GNU_VERNEED",
	elf.SHT_GNU_VERSYM:     "GNU_VERSYM",
}

func translateSectionType(sectionType elf.SectionType) string {
	if found, ok := sectionNames[sectionType]; ok {
		return found
	}
	return "UNKNOWN"
}

func translateSectionFlags(flags elf.SectionFlag) string {
	active := []string{}
	if flags&elf.SHF_WRITE > 0 {
		active = append(active, "WRITE")
	}
	if flags&elf.SHF_ALLOC > 0 {
		active = append(active, "ALLOC")
	}
	if flags&elf.SHF_EXECINSTR > 0 {
		active = append(active, "EXECINSTR")
	}
	if flags&elf.SHF_MERGE > 0 {
		active = append(active, "MERGE")
	}
	if flags&elf.SHF_STRINGS > 0 {
		active = append(active, "STRINGS")
	}
	if flags&elf.SHF_INFO_LINK > 0 {
		active = append(active, "INFO_LINK")
	}
	if flags&elf.SHF_LINK_ORDER > 0 {
		active = append(active, "LINK_ORDER")
	}
	if flags&elf.SHF_OS_NONCONFORMING > 0 {
		active = append(active, "OS_NONCONFORMING")
	}
	if flags&elf.SHF_GROUP > 0 {
		active = append(active, "GROUP")
	}
	if flags&elf.SHF_TLS > 0 {
		active = append(active, "TLS")
	}
	if flags&elf.SHF_COMPRESSED > 0 {
		active = append(active, "COMPRESSED")
	}
	if flags&elf.SHF_MASKOS > 0 {
		active = append(active, "MASKOS")
	}
	if flags&elf.SHF_MASKPROC > 0 {
		active = append(active, "MASKPROC")
	}
	if len(active) == 0 {
		return "-"
	}
	return strings.Join(active, " | ")
}
