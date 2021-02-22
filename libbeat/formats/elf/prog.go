package elf

import "debug/elf"

const (
	// https://refspecs.linuxbase.org/LSB_3.1.1/LSB-Core-generic/LSB-Core-generic.html

	// specifies the location and size of the exception handling information as defined by the .eh_frame_hdr section.
	ptGnuEhFrame elf.ProgType = 0x6474e550
	// specifies the permissions on the segment containing the stack and is used to indicate wether the stack should be executable. The absense of this header indicates that the stack will be executable.
	ptGnuStack elf.ProgType = 0x6474e551
	// specifies the location and size of a segment which may be made read-only after relocation shave been processed.
	ptGnuRelro elf.ProgType = 0x6474e552
)

var progNames = map[elf.ProgType]string{
	elf.PT_NULL:    "NULL",
	elf.PT_LOAD:    "LOAD",
	elf.PT_DYNAMIC: "DYNAMIC",
	elf.PT_INTERP:  "INTERP",
	elf.PT_NOTE:    "NOTE",
	elf.PT_SHLIB:   "SHLIB",
	elf.PT_PHDR:    "PHDR",
	elf.PT_TLS:     "TLS",
	ptGnuEhFrame:   "GNU_EH_FRAME",
	ptGnuStack:     "GNU_STACK",
	ptGnuRelro:     "GNU_RELRO",
}

func translateProgType(progType elf.ProgType) string {
	if found, ok := progNames[progType]; ok {
		return found
	}
	return "UNKNOWN"
}
