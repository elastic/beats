// Created by cgo -godefs - DO NOT EDIT
// cgo -godefs defs_audit_arches.go

package auparse

import "fmt"

type auditArch uint32

const (
	AUDIT_ARCH_AARCH64     auditArch = 0xc00000b7
	AUDIT_ARCH_ARM         auditArch = 0x40000028
	AUDIT_ARCH_ARMEB       auditArch = 0x28
	AUDIT_ARCH_CRIS        auditArch = 0x4000004c
	AUDIT_ARCH_FRV         auditArch = 0x5441
	AUDIT_ARCH_I386        auditArch = 0x40000003
	AUDIT_ARCH_IA64        auditArch = 0xc0000032
	AUDIT_ARCH_M32R        auditArch = 0x58
	AUDIT_ARCH_M68K        auditArch = 0x4
	AUDIT_ARCH_MIPS        auditArch = 0x8
	AUDIT_ARCH_MIPS64      auditArch = 0x80000008
	AUDIT_ARCH_MIPS64N32   auditArch = 0xa0000008
	AUDIT_ARCH_MIPSEL      auditArch = 0x40000008
	AUDIT_ARCH_MIPSEL64    auditArch = 0xc0000008
	AUDIT_ARCH_MIPSEL64N32 auditArch = 0xe0000008
	AUDIT_ARCH_PARISC      auditArch = 0xf
	AUDIT_ARCH_PARISC64    auditArch = 0x8000000f
	AUDIT_ARCH_PPC         auditArch = 0x14
	AUDIT_ARCH_PPC64       auditArch = 0x80000015
	AUDIT_ARCH_PPC64LE     auditArch = 0xc0000015
	AUDIT_ARCH_S390        auditArch = 0x16
	AUDIT_ARCH_S390X       auditArch = 0x80000016
	AUDIT_ARCH_SH          auditArch = 0x2a
	AUDIT_ARCH_SH64        auditArch = 0x8000002a
	AUDIT_ARCH_SHEL        auditArch = 0x4000002a
	AUDIT_ARCH_SHEL64      auditArch = 0xc000002a
	AUDIT_ARCH_SPARC       auditArch = 0x2
	AUDIT_ARCH_SPARC64     auditArch = 0x8000002b
	AUDIT_ARCH_X86_64      auditArch = 0xc000003e
)

var auditArchNames = map[auditArch]string{
	AUDIT_ARCH_AARCH64:     "aarch64",
	AUDIT_ARCH_ARM:         "arm",
	AUDIT_ARCH_ARMEB:       "armeb",
	AUDIT_ARCH_CRIS:        "cris",
	AUDIT_ARCH_FRV:         "frv",
	AUDIT_ARCH_I386:        "i386",
	AUDIT_ARCH_IA64:        "ia64",
	AUDIT_ARCH_M32R:        "m32r",
	AUDIT_ARCH_M68K:        "m68k",
	AUDIT_ARCH_MIPS:        "mips",
	AUDIT_ARCH_MIPS64:      "mips64",
	AUDIT_ARCH_MIPS64N32:   "mips64n32",
	AUDIT_ARCH_MIPSEL:      "mipsel",
	AUDIT_ARCH_MIPSEL64:    "mipsel64",
	AUDIT_ARCH_MIPSEL64N32: "mipsel64n32",
	AUDIT_ARCH_PARISC:      "parisc",
	AUDIT_ARCH_PARISC64:    "parisc64",
	AUDIT_ARCH_PPC:         "ppc",
	AUDIT_ARCH_PPC64:       "ppc64",
	AUDIT_ARCH_PPC64LE:     "ppc64le",
	AUDIT_ARCH_S390:        "s390",
	AUDIT_ARCH_S390X:       "s390x",
	AUDIT_ARCH_SH:          "sh",
	AUDIT_ARCH_SH64:        "sh64",
	AUDIT_ARCH_SHEL:        "shel",
	AUDIT_ARCH_SHEL64:      "shel64",
	AUDIT_ARCH_SPARC:       "sparc",
	AUDIT_ARCH_SPARC64:     "sparc64",
	AUDIT_ARCH_X86_64:      "x86_64",
}

func (a auditArch) String() string {
	name, found := auditArchNames[a]
	if found {
		return name
	}

	return fmt.Sprintf("unknown[%x]", uint32(a))
}
