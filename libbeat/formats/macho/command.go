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

func translateLoadType(loadType uint32) string {
	switch loadType {
	case 0x80000000:
		return "LC_REQ_DYLD"
	case 0x80000018:
		return "LC_LOAD_WEAK_DYLIB"
	case 0x8000001c:
		return "LC_RPATH"
	case 0x8000001f:
		return "LC_REEXPORT_DYLIB"
	case 0x80000022:
		return "LC_DYLD_INFO_ONLY"
	case 0x80000023:
		return "LC_LOAD_UPWARD_DYLIB"
	case 0x80000028:
		return "LC_MAIN"
	case 0x1:
		return "LC_SEGMENT"
	case 0x2:
		return "LC_SYMTAB"
	case 0x3:
		return "LC_SYMSEG"
	case 0x4:
		return "LC_THREAD"
	case 0x5:
		return "LC_UNIXTHREAD"
	case 0x6:
		return "LC_LOADFVMLIB"
	case 0x7:
		return "LC_IDFVMLIB"
	case 0x8:
		return "LC_IDENT"
	case 0x9:
		return "LC_FVMFILE"
	case 0xa:
		return "LC_PREPAGE"
	case 0xb:
		return "LC_DYSYMTAB"
	case 0xc:
		return "LC_LOAD_DYLIB"
	case 0xd:
		return "LC_ID_DYLIB"
	case 0xe:
		return "LC_LOAD_DYLINKER"
	case 0xf:
		return "LC_ID_DYLINKER"
	case 0x10:
		return "LC_PREBOUND_DYLIB"
	case 0x11:
		return "LC_ROUTINES"
	case 0x12:
		return "LC_SUB_FRAMEWORK"
	case 0x13:
		return "LC_SUB_UMBRELLA"
	case 0x14:
		return "LC_SUB_CLIENT"
	case 0x15:
		return "LC_SUB_LIBRARY"
	case 0x16:
		return "LC_TWOLEVEL_HINTS"
	case 0x17:
		return "LC_PREBIND_CKSUM"
	case 0x19:
		return "LC_SEGMENT_64"
	case 0x1a:
		return "LC_ROUTINES_64"
	case 0x1b:
		return "LC_UUID"
	case 0x1d:
		return "LC_CODE_SIGNATURE"
	case 0x1e:
		return "LC_SEGMENT_SPLIT_INFO"
	case 0x20:
		return "LC_LAZY_LOAD_DYLIB"
	case 0x21:
		return "LC_ENCRYPTION_INFO"
	case 0x22:
		return "LC_DYLD_INFO"
	case 0x24:
		return "LC_VERSION_MIN_MACOSX"
	case 0x25:
		return "LC_VERSION_MIN_IPHONEOS"
	case 0x26:
		return "LC_FUNCTION_STARTS"
	case 0x27:
		return "LC_DYLD_ENVIRONMENT"
	case 0x29:
		return "LC_DATA_IN_CODE"
	case 0x2A:
		return "LC_SOURCE_VERSION"
	case 0x2B:
		return "LC_DYLIB_CODE_SIGN_DRS"
	case 0x2C:
		return "LC_ENCRYPTION_INFO_64"
	case 0x2D:
		return "LC_LINKER_OPTION"
	case 0x2E:
		return "LC_LINKER_OPTIMIZATION_HINT"
	default:
		return "LC_UNKNOWN"
	}
}

func loadCommands(f *macho.File) []Command {
	commands := make([]Command, len(f.Loads))
	for i, load := range f.Loads {
		data := load.Raw()
		loadType := f.ByteOrder.Uint32(data[0:4])
		command := Command{
			Number: int64(loadType),
			Size:   int64(f.ByteOrder.Uint32(data[4:8])),
		}
		command.Type = translateLoadType(loadType)
		commands[i] = command
	}
	return commands
}
