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

import "debug/macho"

const (
	CPU_SUBTYPE_MASK  uint32 = 0x00ffffff
	CPU_ARCH_ABI64    uint32 = 0x01000000
	CPU_ARCH_ABI64_32 uint32 = 0x02000000

	// cpu types
	CPU_TYPE_VAX       uint32 = 1
	CPU_TYPE_MC680X0   uint32 = 6
	CPU_TYPE_X86       uint32 = 7
	CPU_TYPE_I386      uint32 = CPU_TYPE_X86
	CPU_TYPE_X86_64    uint32 = CPU_TYPE_X86 | CPU_ARCH_ABI64
	CPU_TYPE_MIPS      uint32 = 8
	CPU_TYPE_MC98000   uint32 = 10
	CPU_TYPE_HPPA      uint32 = 11
	CPU_TYPE_ARM       uint32 = 12
	CPU_TYPE_ARM64     uint32 = CPU_TYPE_ARM | CPU_ARCH_ABI64
	CPU_TYPE_ARM64_32  uint32 = CPU_TYPE_ARM | CPU_ARCH_ABI64_32
	CPU_TYPE_MC88000   uint32 = 13
	CPU_TYPE_SPARC     uint32 = 14
	CPU_TYPE_I860      uint32 = 15
	CPU_TYPE_ALPHA     uint32 = 16
	CPU_TYPE_POWERPC   uint32 = 18
	CPU_TYPE_POWERPC64 uint32 = CPU_TYPE_POWERPC | CPU_ARCH_ABI64

	// cpu sub-types
	CPU_SUBTYPE_LITTLE_ENDIAN    uint32 = 0
	CPU_SUBTYPE_BIG_ENDIAN       uint32 = 1
	CPU_SUBTYPE_VAX_ALL          uint32 = 0
	CPU_SUBTYPE_VAX780           uint32 = 1
	CPU_SUBTYPE_VAX785           uint32 = 2
	CPU_SUBTYPE_VAX750           uint32 = 3
	CPU_SUBTYPE_VAX730           uint32 = 4
	CPU_SUBTYPE_UVAXI            uint32 = 5
	CPU_SUBTYPE_UVAXII           uint32 = 6
	CPU_SUBTYPE_VAX8200          uint32 = 7
	CPU_SUBTYPE_VAX8500          uint32 = 8
	CPU_SUBTYPE_VAX8600          uint32 = 9
	CPU_SUBTYPE_VAX8650          uint32 = 10
	CPU_SUBTYPE_VAX8800          uint32 = 11
	CPU_SUBTYPE_UVAXIII          uint32 = 12
	CPU_SUBTYPE_MC680X0_ALL      uint32 = 1
	CPU_SUBTYPE_MC68030          uint32 = 1
	CPU_SUBTYPE_MC68040          uint32 = 2
	CPU_SUBTYPE_MC68030_ONLY     uint32 = 3
	CPU_SUBTYPE_I386_ALL         uint32 = 3
	CPU_SUBTYPE_386              uint32 = 3
	CPU_SUBTYPE_486              uint32 = 4
	CPU_SUBTYPE_486SX            uint32 = 4 + (8 << 4)
	CPU_SUBTYPE_586              uint32 = 5
	CPU_SUBTYPE_PENT             uint32 = 5
	CPU_SUBTYPE_PENTPRO          uint32 = 6 + (1 << 4)
	CPU_SUBTYPE_PENTII_M3        uint32 = 6 + (3 << 4)
	CPU_SUBTYPE_PENTII_M5        uint32 = 6 + (5 << 4)
	CPU_SUBTYPE_CELERON          uint32 = 7 + (6 << 4)
	CPU_SUBTYPE_CELERON_MOBILE   uint32 = 7 + (7 << 4)
	CPU_SUBTYPE_PENTIUM_3        uint32 = 8
	CPU_SUBTYPE_PENTIUM_3_M      uint32 = 8 + (1 << 4)
	CPU_SUBTYPE_PENTIUM_3_XEON   uint32 = 8 + (2 << 4)
	CPU_SUBTYPE_PENTIUM_M        uint32 = 9
	CPU_SUBTYPE_PENTIUM_4        uint32 = 10
	CPU_SUBTYPE_PENTIUM_4_M      uint32 = 10 + (1 << 4)
	CPU_SUBTYPE_ITANIUM          uint32 = 11
	CPU_SUBTYPE_ITANIUM_2        uint32 = 11 + (1 << 4)
	CPU_SUBTYPE_XEON             uint32 = 12
	CPU_SUBTYPE_XEON_MP          uint32 = 12 + (1 << 4)
	CPU_SUBTYPE_INTEL_FAMILY_MAX uint32 = 15
	CPU_SUBTYPE_INTEL_MODEL_ALL  uint32 = 0
	CPU_SUBTYPE_X86_ALL          uint32 = 3
	CPU_SUBTYPE_X86_64_ALL       uint32 = 3
	CPU_SUBTYPE_X86_ARCH1        uint32 = 4
	CPU_SUBTYPE_X86_64_H         uint32 = 8
	CPU_SUBTYPE_MIPS_ALL         uint32 = 0
	CPU_SUBTYPE_MIPS_R2300       uint32 = 1
	CPU_SUBTYPE_MIPS_R2600       uint32 = 2
	CPU_SUBTYPE_MIPS_R2800       uint32 = 3
	CPU_SUBTYPE_MIPS_R2000A      uint32 = 4
	CPU_SUBTYPE_MIPS_R2000       uint32 = 5
	CPU_SUBTYPE_MIPS_R3000A      uint32 = 6
	CPU_SUBTYPE_MIPS_R3000       uint32 = 7
	CPU_SUBTYPE_MC98000_ALL      uint32 = 0
	CPU_SUBTYPE_MC98601          uint32 = 1
	CPU_SUBTYPE_HPPA_ALL         uint32 = 0
	CPU_SUBTYPE_HPPA_7100        uint32 = 0
	CPU_SUBTYPE_HPPA_7100LC      uint32 = 1
	CPU_SUBTYPE_MC88000_ALL      uint32 = 0
	CPU_SUBTYPE_MC88100          uint32 = 1
	CPU_SUBTYPE_MC88110          uint32 = 2
	CPU_SUBTYPE_SPARC_ALL        uint32 = 0
	CPU_SUBTYPE_I860_ALL         uint32 = 0
	CPU_SUBTYPE_I860_860         uint32 = 1
	CPU_SUBTYPE_POWERPC_ALL      uint32 = 0
	CPU_SUBTYPE_POWERPC_601      uint32 = 1
	CPU_SUBTYPE_POWERPC_602      uint32 = 2
	CPU_SUBTYPE_POWERPC_603      uint32 = 3
	CPU_SUBTYPE_POWERPC_603E     uint32 = 4
	CPU_SUBTYPE_POWERPC_603EV    uint32 = 5
	CPU_SUBTYPE_POWERPC_604      uint32 = 6
	CPU_SUBTYPE_POWERPC_604E     uint32 = 7
	CPU_SUBTYPE_POWERPC_620      uint32 = 8
	CPU_SUBTYPE_POWERPC_750      uint32 = 9
	CPU_SUBTYPE_POWERPC_7400     uint32 = 10
	CPU_SUBTYPE_POWERPC_7450     uint32 = 11
	CPU_SUBTYPE_POWERPC_970      uint32 = 100
	CPU_SUBTYPE_ARM_ALL          uint32 = 0
	CPU_SUBTYPE_ARM_V4T          uint32 = 5
	CPU_SUBTYPE_ARM_V6           uint32 = 6
	CPU_SUBTYPE_ARM_V5TEJ        uint32 = 7
	CPU_SUBTYPE_ARM_XSCALE       uint32 = 8
	CPU_SUBTYPE_ARM_V7           uint32 = 9
	CPU_SUBTYPE_ARM_V7F          uint32 = 10
	CPU_SUBTYPE_ARM_V7S          uint32 = 11
	CPU_SUBTYPE_ARM_V7K          uint32 = 12
	CPU_SUBTYPE_ARM_V6M          uint32 = 14
	CPU_SUBTYPE_ARM_V7M          uint32 = 15
	CPU_SUBTYPE_ARM_V7EM         uint32 = 16
	CPU_SUBTYPE_ARM_V8           uint32 = 13
	CPU_SUBTYPE_ARM64_ALL        uint32 = 0
	CPU_SUBTYPE_ARM64_V8         uint32 = 1
	CPU_SUBTYPE_ARM64_E          uint32 = 2
	CPU_SUBTYPE_ARM64_32_ALL     uint32 = 0
	CPU_SUBTYPE_ARM64_32_V8      uint32 = 1
)

var flagMaps = []struct {
	name       string
	cpuType    uint32
	cpuSubtype uint32
}{
	{"ppc64", CPU_TYPE_POWERPC64, CPU_SUBTYPE_POWERPC_ALL},
	{"x86_64", CPU_TYPE_X86_64, CPU_SUBTYPE_X86_64_ALL},
	{"x86_64h", CPU_TYPE_X86_64, CPU_SUBTYPE_X86_64_H},
	{"arm64", CPU_TYPE_ARM64, CPU_SUBTYPE_ARM64_ALL},
	{"arm64_32", CPU_TYPE_ARM64_32, CPU_SUBTYPE_ARM64_32_ALL},
	{"ppc970-64", CPU_TYPE_POWERPC64, CPU_SUBTYPE_POWERPC_970},
	{"ppc", CPU_TYPE_POWERPC, CPU_SUBTYPE_POWERPC_ALL},
	{"i386", CPU_TYPE_I386, CPU_SUBTYPE_I386_ALL},
	{"m68k", CPU_TYPE_MC680X0, CPU_SUBTYPE_MC680X0_ALL},
	{"hppa", CPU_TYPE_HPPA, CPU_SUBTYPE_HPPA_ALL},
	{"sparc", CPU_TYPE_SPARC, CPU_SUBTYPE_SPARC_ALL},
	{"m88k", CPU_TYPE_MC88000, CPU_SUBTYPE_MC88000_ALL},
	{"i860", CPU_TYPE_I860, CPU_SUBTYPE_I860_ALL},
	{"arm", CPU_TYPE_ARM, CPU_SUBTYPE_ARM_ALL},
	{"ppc601", CPU_TYPE_POWERPC, CPU_SUBTYPE_POWERPC_601},
	{"ppc603", CPU_TYPE_POWERPC, CPU_SUBTYPE_POWERPC_603},
	{"ppc603e", CPU_TYPE_POWERPC, CPU_SUBTYPE_POWERPC_603E},
	{"ppc603ev", CPU_TYPE_POWERPC, CPU_SUBTYPE_POWERPC_603EV},
	{"ppc604", CPU_TYPE_POWERPC, CPU_SUBTYPE_POWERPC_604},
	{"ppc604e", CPU_TYPE_POWERPC, CPU_SUBTYPE_POWERPC_604E},
	{"ppc750", CPU_TYPE_POWERPC, CPU_SUBTYPE_POWERPC_750},
	{"ppc7400", CPU_TYPE_POWERPC, CPU_SUBTYPE_POWERPC_7400},
	{"ppc7450", CPU_TYPE_POWERPC, CPU_SUBTYPE_POWERPC_7450},
	{"ppc970", CPU_TYPE_POWERPC, CPU_SUBTYPE_POWERPC_970},
	{"i486", CPU_TYPE_I386, CPU_SUBTYPE_486},
	{"i486SX", CPU_TYPE_I386, CPU_SUBTYPE_486SX},
	{"i586", CPU_TYPE_I386, CPU_SUBTYPE_586},
	{"i686", CPU_TYPE_I386, CPU_SUBTYPE_PENTPRO},
	{"pentIIm3", CPU_TYPE_I386, CPU_SUBTYPE_PENTII_M3},
	{"pentIIm5", CPU_TYPE_I386, CPU_SUBTYPE_PENTII_M5},
	{"pentium4", CPU_TYPE_I386, CPU_SUBTYPE_PENTIUM_4},
	{"m68030", CPU_TYPE_MC680X0, CPU_SUBTYPE_MC68030_ONLY},
	{"m68040", CPU_TYPE_MC680X0, CPU_SUBTYPE_MC68040},
	{"hppa7100LC", CPU_TYPE_HPPA, CPU_SUBTYPE_HPPA_7100LC},
	{"armv4t", CPU_TYPE_ARM, CPU_SUBTYPE_ARM_V4T},
	{"armv5", CPU_TYPE_ARM, CPU_SUBTYPE_ARM_V5TEJ},
	{"xscale", CPU_TYPE_ARM, CPU_SUBTYPE_ARM_XSCALE},
	{"armv6", CPU_TYPE_ARM, CPU_SUBTYPE_ARM_V6},
	{"armv6m", CPU_TYPE_ARM, CPU_SUBTYPE_ARM_V6M},
	{"armv7", CPU_TYPE_ARM, CPU_SUBTYPE_ARM_V7},
	{"armv7f", CPU_TYPE_ARM, CPU_SUBTYPE_ARM_V7F},
	{"armv7s", CPU_TYPE_ARM, CPU_SUBTYPE_ARM_V7S},
	{"armv7k", CPU_TYPE_ARM, CPU_SUBTYPE_ARM_V7K},
	{"armv7m", CPU_TYPE_ARM, CPU_SUBTYPE_ARM_V7M},
	{"armv7em", CPU_TYPE_ARM, CPU_SUBTYPE_ARM_V7EM},
	{"arm64v8", CPU_TYPE_ARM64, CPU_SUBTYPE_ARM64_V8},
	{"arm64e", CPU_TYPE_ARM64, CPU_SUBTYPE_ARM64_E},
	{"arm64_32_v8", CPU_TYPE_ARM64_32, CPU_SUBTYPE_ARM64_32_V8},
	// others
	{"pentium", CPU_TYPE_I386, CPU_SUBTYPE_PENT},
	{"pentpro", CPU_TYPE_I386, CPU_SUBTYPE_PENTPRO},
	{"x86", CPU_TYPE_I386, CPU_SUBTYPE_I386_ALL},
}

// the default string translations are gross
func translateCPU(cpu macho.Cpu, subtype uint32) string {
	cputype := uint32(cpu)
	for _, cpuMapping := range flagMaps {
		if cpuMapping.cpuType == cputype && cpuMapping.cpuSubtype == (CPU_SUBTYPE_MASK&subtype) {
			return cpuMapping.name
		}
	}
	return "unknown"
}
