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

// /// Get the cputype and cpusubtype from a name
// pub fn get_arch_from_flag(name: &str) -> Option<(CpuType, CpuSubType)> {
//     get_arch_from_flag_no_alias(name).or_else(|| {
//         // we also handle some common aliases
//         match name {
//             // these are used by apple
//             "pentium" => Some((CPU_TYPE_I386, CPU_SUBTYPE_PENT)),
//             "pentpro" => Some((CPU_TYPE_I386, CPU_SUBTYPE_PENTPRO)),
//             // these are used commonly for consistency
//             "x86" => Some((CPU_TYPE_I386, CPU_SUBTYPE_I386_ALL)),
//             _ => None,
//         }
//     })
// }

// /// An alias for u32
// pub type CpuType = u32;
// /// An alias for u32
// pub type CpuSubType = u32;

// /// the mask for CPU feature flags
// pub const CPU_SUBTYPE_MASK: u32 = 0xff00_0000;
// /// mask for architecture bits
// pub const CPU_ARCH_MASK: CpuType = 0xff00_0000;
// /// the mask for 64 bit ABI
// pub const CPU_ARCH_ABI64: CpuType = 0x0100_0000;
// /// the mask for ILP32 ABI on 64 bit hardware
// pub const CPU_ARCH_ABI64_32: CpuType = 0x0200_0000;

// // CPU Types
// pub const CPU_TYPE_ANY: CpuType = !0;
// pub const CPU_TYPE_VAX: CpuType = 1;
// pub const CPU_TYPE_MC680X0: CpuType = 6;
// pub const CPU_TYPE_X86: CpuType = 7;
// pub const CPU_TYPE_I386: CpuType = CPU_TYPE_X86;
// pub const CPU_TYPE_X86_64: CpuType = CPU_TYPE_X86 | CPU_ARCH_ABI64;
// pub const CPU_TYPE_MIPS: CpuType = 8;
// pub const CPU_TYPE_MC98000: CpuType = 10;
// pub const CPU_TYPE_HPPA: CpuType = 11;
// pub const CPU_TYPE_ARM: CpuType = 12;
// pub const CPU_TYPE_ARM64: CpuType = CPU_TYPE_ARM | CPU_ARCH_ABI64;
// pub const CPU_TYPE_ARM64_32: CpuType = CPU_TYPE_ARM | CPU_ARCH_ABI64_32;
// pub const CPU_TYPE_MC88000: CpuType = 13;
// pub const CPU_TYPE_SPARC: CpuType = 14;
// pub const CPU_TYPE_I860: CpuType = 15;
// pub const CPU_TYPE_ALPHA: CpuType = 16;
// pub const CPU_TYPE_POWERPC: CpuType = 18;
// pub const CPU_TYPE_POWERPC64: CpuType = CPU_TYPE_POWERPC | CPU_ARCH_ABI64;

// // CPU Subtypes
// pub const CPU_SUBTYPE_MULTIPLE: CpuSubType = !0;
// pub const CPU_SUBTYPE_LITTLE_ENDIAN: CpuSubType = 0;
// pub const CPU_SUBTYPE_BIG_ENDIAN: CpuSubType = 1;
// pub const CPU_SUBTYPE_VAX_ALL: CpuSubType = 0;
// pub const CPU_SUBTYPE_VAX780: CpuSubType = 1;
// pub const CPU_SUBTYPE_VAX785: CpuSubType = 2;
// pub const CPU_SUBTYPE_VAX750: CpuSubType = 3;
// pub const CPU_SUBTYPE_VAX730: CpuSubType = 4;
// pub const CPU_SUBTYPE_UVAXI: CpuSubType = 5;
// pub const CPU_SUBTYPE_UVAXII: CpuSubType = 6;
// pub const CPU_SUBTYPE_VAX8200: CpuSubType = 7;
// pub const CPU_SUBTYPE_VAX8500: CpuSubType = 8;
// pub const CPU_SUBTYPE_VAX8600: CpuSubType = 9;
// pub const CPU_SUBTYPE_VAX8650: CpuSubType = 10;
// pub const CPU_SUBTYPE_VAX8800: CpuSubType = 11;
// pub const CPU_SUBTYPE_UVAXIII: CpuSubType = 12;
// pub const CPU_SUBTYPE_MC680X0_ALL: CpuSubType = 1;
// pub const CPU_SUBTYPE_MC68030: CpuSubType = 1; /* compat */
// pub const CPU_SUBTYPE_MC68040: CpuSubType = 2;
// pub const CPU_SUBTYPE_MC68030_ONLY: CpuSubType = 3;

// macro_rules! CPU_SUBTYPE_INTEL {
//     ($f:expr, $m:expr) => {{
//         ($f) + (($m) << 4)
//     }};
// }

// pub const CPU_SUBTYPE_I386_ALL: CpuSubType = CPU_SUBTYPE_INTEL!(3, 0);
// pub const CPU_SUBTYPE_386: CpuSubType = CPU_SUBTYPE_INTEL!(3, 0);
// pub const CPU_SUBTYPE_486: CpuSubType = CPU_SUBTYPE_INTEL!(4, 0);
// pub const CPU_SUBTYPE_486SX: CpuSubType = CPU_SUBTYPE_INTEL!(4, 8); // 8 << 4 = 128
// pub const CPU_SUBTYPE_586: CpuSubType = CPU_SUBTYPE_INTEL!(5, 0);
// pub const CPU_SUBTYPE_PENT: CpuSubType = CPU_SUBTYPE_INTEL!(5, 0);
// pub const CPU_SUBTYPE_PENTPRO: CpuSubType = CPU_SUBTYPE_INTEL!(6, 1);
// pub const CPU_SUBTYPE_PENTII_M3: CpuSubType = CPU_SUBTYPE_INTEL!(6, 3);
// pub const CPU_SUBTYPE_PENTII_M5: CpuSubType = CPU_SUBTYPE_INTEL!(6, 5);
// pub const CPU_SUBTYPE_CELERON: CpuSubType = CPU_SUBTYPE_INTEL!(7, 6);
// pub const CPU_SUBTYPE_CELERON_MOBILE: CpuSubType = CPU_SUBTYPE_INTEL!(7, 7);
// pub const CPU_SUBTYPE_PENTIUM_3: CpuSubType = CPU_SUBTYPE_INTEL!(8, 0);
// pub const CPU_SUBTYPE_PENTIUM_3_M: CpuSubType = CPU_SUBTYPE_INTEL!(8, 1);
// pub const CPU_SUBTYPE_PENTIUM_3_XEON: CpuSubType = CPU_SUBTYPE_INTEL!(8, 2);
// pub const CPU_SUBTYPE_PENTIUM_M: CpuSubType = CPU_SUBTYPE_INTEL!(9, 0);
// pub const CPU_SUBTYPE_PENTIUM_4: CpuSubType = CPU_SUBTYPE_INTEL!(10, 0);
// pub const CPU_SUBTYPE_PENTIUM_4_M: CpuSubType = CPU_SUBTYPE_INTEL!(10, 1);
// pub const CPU_SUBTYPE_ITANIUM: CpuSubType = CPU_SUBTYPE_INTEL!(11, 0);
// pub const CPU_SUBTYPE_ITANIUM_2: CpuSubType = CPU_SUBTYPE_INTEL!(11, 1);
// pub const CPU_SUBTYPE_XEON: CpuSubType = CPU_SUBTYPE_INTEL!(12, 0);
// pub const CPU_SUBTYPE_XEON_MP: CpuSubType = CPU_SUBTYPE_INTEL!(12, 1);
// pub const CPU_SUBTYPE_INTEL_FAMILY_MAX: CpuSubType = 15;
// pub const CPU_SUBTYPE_INTEL_MODEL_ALL: CpuSubType = 0;
// pub const CPU_SUBTYPE_X86_ALL: CpuSubType = 3;
// pub const CPU_SUBTYPE_X86_64_ALL: CpuSubType = 3;
// pub const CPU_SUBTYPE_X86_ARCH1: CpuSubType = 4;
// pub const CPU_SUBTYPE_X86_64_H: CpuSubType = 8;
// pub const CPU_SUBTYPE_MIPS_ALL: CpuSubType = 0;
// pub const CPU_SUBTYPE_MIPS_R2300: CpuSubType = 1;
// pub const CPU_SUBTYPE_MIPS_R2600: CpuSubType = 2;
// pub const CPU_SUBTYPE_MIPS_R2800: CpuSubType = 3;
// pub const CPU_SUBTYPE_MIPS_R2000A: CpuSubType = 4;
// pub const CPU_SUBTYPE_MIPS_R2000: CpuSubType = 5;
// pub const CPU_SUBTYPE_MIPS_R3000A: CpuSubType = 6;
// pub const CPU_SUBTYPE_MIPS_R3000: CpuSubType = 7;
// pub const CPU_SUBTYPE_MC98000_ALL: CpuSubType = 0;
// pub const CPU_SUBTYPE_MC98601: CpuSubType = 1;
// pub const CPU_SUBTYPE_HPPA_ALL: CpuSubType = 0;
// pub const CPU_SUBTYPE_HPPA_7100: CpuSubType = 0;
// pub const CPU_SUBTYPE_HPPA_7100LC: CpuSubType = 1;
// pub const CPU_SUBTYPE_MC88000_ALL: CpuSubType = 0;
// pub const CPU_SUBTYPE_MC88100: CpuSubType = 1;
// pub const CPU_SUBTYPE_MC88110: CpuSubType = 2;
// pub const CPU_SUBTYPE_SPARC_ALL: CpuSubType = 0;
// pub const CPU_SUBTYPE_I860_ALL: CpuSubType = 0;
// pub const CPU_SUBTYPE_I860_860: CpuSubType = 1;
// pub const CPU_SUBTYPE_POWERPC_ALL: CpuSubType = 0;
// pub const CPU_SUBTYPE_POWERPC_601: CpuSubType = 1;
// pub const CPU_SUBTYPE_POWERPC_602: CpuSubType = 2;
// pub const CPU_SUBTYPE_POWERPC_603: CpuSubType = 3;
// pub const CPU_SUBTYPE_POWERPC_603E: CpuSubType = 4;
// pub const CPU_SUBTYPE_POWERPC_603EV: CpuSubType = 5;
// pub const CPU_SUBTYPE_POWERPC_604: CpuSubType = 6;
// pub const CPU_SUBTYPE_POWERPC_604E: CpuSubType = 7;
// pub const CPU_SUBTYPE_POWERPC_620: CpuSubType = 8;
// pub const CPU_SUBTYPE_POWERPC_750: CpuSubType = 9;
// pub const CPU_SUBTYPE_POWERPC_7400: CpuSubType = 10;
// pub const CPU_SUBTYPE_POWERPC_7450: CpuSubType = 11;
// pub const CPU_SUBTYPE_POWERPC_970: CpuSubType = 100;
// pub const CPU_SUBTYPE_ARM_ALL: CpuSubType = 0;
// pub const CPU_SUBTYPE_ARM_V4T: CpuSubType = 5;
// pub const CPU_SUBTYPE_ARM_V6: CpuSubType = 6;
// pub const CPU_SUBTYPE_ARM_V5TEJ: CpuSubType = 7;
// pub const CPU_SUBTYPE_ARM_XSCALE: CpuSubType = 8;
// pub const CPU_SUBTYPE_ARM_V7: CpuSubType = 9;
// pub const CPU_SUBTYPE_ARM_V7F: CpuSubType = 10;
// pub const CPU_SUBTYPE_ARM_V7S: CpuSubType = 11;
// pub const CPU_SUBTYPE_ARM_V7K: CpuSubType = 12;
// pub const CPU_SUBTYPE_ARM_V6M: CpuSubType = 14;
// pub const CPU_SUBTYPE_ARM_V7M: CpuSubType = 15;
// pub const CPU_SUBTYPE_ARM_V7EM: CpuSubType = 16;
// pub const CPU_SUBTYPE_ARM_V8: CpuSubType = 13;
// pub const CPU_SUBTYPE_ARM64_ALL: CpuSubType = 0;
// pub const CPU_SUBTYPE_ARM64_V8: CpuSubType = 1;
// pub const CPU_SUBTYPE_ARM64_E: CpuSubType = 2;
// pub const CPU_SUBTYPE_ARM64_32_ALL: CpuSubType = 0;
// pub const CPU_SUBTYPE_ARM64_32_V8: CpuSubType = 1;

// the default string translations are gross
func translateCPU(cpu macho.Cpu) string {
	switch cpu {
	case macho.Cpu386:
		return "x86"
	case macho.CpuAmd64:
		return "x86_64"
	case macho.CpuArm:
		return "arm"
	case macho.CpuArm64:
		return "arm64"
	case macho.CpuPpc:
		return "ppc"
	case macho.CpuPpc64:
		return "ppc64"
	default:
		return "unknown"
	}
}
