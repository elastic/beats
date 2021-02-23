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

func headerFlags(flags uint32) []string {
	flagNames := []string{}
	if (flags & 0x1) > 0 {
		flagNames = append(flagNames, "MH_NOUNDEFS")
	}
	if (flags & 0x2) > 0 {
		flagNames = append(flagNames, "MH_INCRLINK")
	}
	if (flags & 0x4) > 0 {
		flagNames = append(flagNames, "MH_DYLDLINK")
	}
	if (flags & 0x8) > 0 {
		flagNames = append(flagNames, "MH_BINDATLOAD")
	}
	if (flags & 0x10) > 0 {
		flagNames = append(flagNames, "MH_PREBOUND")
	}
	if (flags & 0x20) > 0 {
		flagNames = append(flagNames, "MH_SPLIT_SEGS")
	}
	if (flags & 0x40) > 0 {
		flagNames = append(flagNames, "MH_LAZY_INIT")
	}
	if (flags & 0x80) > 0 {
		flagNames = append(flagNames, "MH_TWOLEVEL")
	}
	if (flags & 0x100) > 0 {
		flagNames = append(flagNames, "MH_FORCE_FLAT")
	}
	if (flags & 0x200) > 0 {
		flagNames = append(flagNames, "MH_NOMULTIDEFS")
	}

	if (flags & 0x400) > 0 {
		flagNames = append(flagNames, "MH_NOFIXPREBINDING")
	}
	if (flags & 0x800) > 0 {
		flagNames = append(flagNames, "MH_PREBINDABLE")
	}
	if (flags & 0x1000) > 0 {
		flagNames = append(flagNames, "MH_ALLMODSBOUND")
	}
	if (flags & 0x2000) > 0 {
		flagNames = append(flagNames, "MH_SUBSECTIONS_VIA_SYMBOLS")
	}
	if (flags & 0x4000) > 0 {
		flagNames = append(flagNames, "MH_CANONICAL")
	}
	if (flags & 0x8000) > 0 {
		flagNames = append(flagNames, "MH_WEAK_DEFINES")
	}
	if (flags & 0x10000) > 0 {
		flagNames = append(flagNames, "MH_BINDS_TO_WEAK")
	}
	if (flags & 0x20000) > 0 {
		flagNames = append(flagNames, "MH_ALLOW_STACK_EXECUTION")
	}
	if (flags & 0x40000) > 0 {
		flagNames = append(flagNames, "MH_ROOT_SAFE")
	}
	if (flags & 0x80000) > 0 {
		flagNames = append(flagNames, "MH_SETUID_SAFE")
	}
	if (flags & 0x100000) > 0 {
		flagNames = append(flagNames, "MH_NO_REEXPORTED_DYLIBS")
	}
	if (flags & 0x200000) > 0 {
		flagNames = append(flagNames, "MH_PIE")
	}
	if (flags & 0x400000) > 0 {
		flagNames = append(flagNames, "MH_DEAD_STRIPPABLE_DYLIB")
	}
	if (flags & 0x800000) > 0 {
		flagNames = append(flagNames, "MH_HAS_TLV_DESCRIPTORS")
	}
	if (flags & 0x1000000) > 0 {
		flagNames = append(flagNames, "MH_NO_HEAP_EXECUTION")
	}
	if (flags & 0x2000000) > 0 {
		flagNames = append(flagNames, "MH_APP_EXTENSION_SAFE")
	}
	return flagNames
}
