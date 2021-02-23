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

func sectionType(flags uint32) string {
	maskedType := flags & 0x000000ff
	switch maskedType {
	case 0x00:
		return "S_REGULAR"
	case 0x01:
		return "S_ZEROFILL"
	case 0x02:
		return "S_CSTRING_LITERALS"
	case 0x03:
		return "S_4BYTE_LITERALS"
	case 0x04:
		return "S_8BYTE_LITERALS"
	case 0x05:
		return "S_LITERAL_POINTERS"
	case 0x06:
		return "S_NON_LAZY_SYMBOL_POINTERS"
	case 0x07:
		return "S_LAZY_SYMBOL_POINTERS"
	case 0x08:
		return "S_SYMBOL_STUBS"
	case 0x09:
		return "S_MOD_INIT_FUNC_POINTERS"
	case 0x0a:
		return "S_MOD_TERM_FUNC_POINTERS"
	case 0x0b:
		return "S_COALESCED"
	case 0x0c:
		return "S_GB_ZEROFILL"
	case 0x0d:
		return "S_INTERPOSING"
	case 0x0e:
		return "S_16BYTE_LITERALS"
	case 0x0f:
		return "S_DTRACE_DOF"
	case 0x10:
		return "S_LAZY_DYLIB_SYMBOL_POINTERS"
	case 0x11:
		return "S_THREAD_LOCAL_REGULAR"
	case 0x12:
		return "S_THREAD_LOCAL_ZEROFILL"
	case 0x13:
		return "S_THREAD_LOCAL_VARIABLES"
	case 0x14:
		return "S_THREAD_LOCAL_VARIABLE_POINTERS"
	case 0x15:
		return "S_THREAD_LOCAL_INIT_FUNCTION_POINTERS"
	default:
		return "UNKNOWN"
	}
}

func sectionFlags(flags uint32) []string {
	flagNames := []string{}
	if (flags & 0x80000000) > 0 {
		flagNames = append(flagNames, "S_ATTR_PURE_INSTRUCTIONS")
	}
	if (flags & 0x40000000) > 0 {
		flagNames = append(flagNames, "S_ATTR_NO_TOC")
	}
	if (flags & 0x20000000) > 0 {
		flagNames = append(flagNames, "S_ATTR_STRIP_STATIC_SYMS")
	}
	if (flags & 0x10000000) > 0 {
		flagNames = append(flagNames, "S_ATTR_NO_DEAD_STRIP")
	}
	if (flags & 0x08000000) > 0 {
		flagNames = append(flagNames, "S_ATTR_LIVE_SUPPORT")
	}
	if (flags & 0x04000000) > 0 {
		flagNames = append(flagNames, "S_ATTR_SELF_MODIFYING_CODE")
	}
	if (flags & 0x02000000) > 0 {
		flagNames = append(flagNames, "S_ATTR_DEBUG")
	}
	if (flags & 0x00000400) > 0 {
		flagNames = append(flagNames, "S_ATTR_SOME_INSTRUCTIONS")
	}
	if (flags & 0x00000200) > 0 {
		flagNames = append(flagNames, "S_ATTR_EXT_RELOC")
	}
	if (flags & 0x00000100) > 0 {
		flagNames = append(flagNames, "S_ATTR_LOC_RELOC")
	}
	return flagNames
}
