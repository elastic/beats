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

package elf

import (
	"debug/elf"
	"errors"
	"io/ioutil"
	"regexp"
	"sort"
	"strings"

	"github.com/knightsc/gapstone"
)

var (
	exclusionsRegex = []*regexp.Regexp{
		regexp.MustCompile(`^[_\.].*$`), // Function names starting with . or _
		regexp.MustCompile(`^.*64$`),    // x64-64 specific functions
		regexp.MustCompile(`^str.*$`),   // gcc significantly changes string functions depending on the target architecture, so we ignore them
		regexp.MustCompile(`^mem.*$`),   // gcc significantly changes string functions depending on the target architecture, so we ignore them
	}
	exclusionsString = []string{
		"__libc_start_main", // main function
		"main",              // main function	z
		"abort",             // ARM default
		"cachectl",          // MIPS default
		"cacheflush",        // MIPS default
		"puts",              // Compiler optimization (function replacement)
		"atol",              // Compiler optimization (function replacement)
		"malloc_trim",       // GNU extensions
	}
)

func canExclude(symbol elf.Symbol) bool {
	if elf.ST_TYPE(symbol.Info) != elf.STT_FUNC {
		return true
	}
	if elf.ST_BIND(symbol.Info) != elf.STB_GLOBAL {
		return true
	}
	if elf.ST_VISIBILITY(symbol.Other) != elf.STV_DEFAULT {
		return true
	}
	if symbol.Name == "" {
		return true
	}

	for _, exclusion := range exclusionsString {
		if symbol.Name == exclusion {
			return true
		}
	}
	for _, exclusion := range exclusionsRegex {
		if exclusion.MatchString(symbol.Name) {
			return true
		}
	}
	return false
}

func capstoneArgs(f *elf.File) (int, int, bool) {
	switch {
	case f.Class == elf.ELFCLASS32 && f.Machine == elf.EM_386:
		return gapstone.CS_ARCH_X86, gapstone.CS_MODE_32, true
	case f.Class == elf.ELFCLASS64 && f.Machine == elf.EM_X86_64:
		return gapstone.CS_ARCH_X86, gapstone.CS_MODE_64, true
	case f.Class == elf.ELFCLASS32 && f.Machine == elf.EM_ARM:
		return gapstone.CS_ARCH_ARM, gapstone.CS_MODE_ARM, true
	case f.Class == elf.ELFCLASS32 && f.Machine == elf.EM_MIPS:
		return gapstone.CS_ARCH_MIPS, int(gapstone.CS_MODE_MIPS32) | gapstone.CS_MODE_BIG_ENDIAN, true
	default:
		return 0, 0, false
	}
}

func isX86(f *elf.File) bool {
	return (f.Class == elf.ELFCLASS64 && f.Machine == elf.EM_X86_64) || (f.Class == elf.ELFCLASS32 && f.Machine == elf.EM_386)
}

func stringMember(ary []string, test string) bool {
	for _, a := range ary {
		if a == test {
			return true
		}
	}
	return false
}

func getImageBase(f *elf.File) uint64 {
	for _, segment := range f.Progs {
		if segment.Type == elf.PT_LOAD {
			return segment.Vaddr
		}
	}
	return 0
}

func extractCallDestinations(f *elf.File) ([]string, error) {
	arch, mode, found := capstoneArgs(f)
	if !found {
		return nil, nil
	}
	entryPoint := f.Entry
	var offset uint64
	var err error
	var data []byte
	for _, section := range f.Sections {
		if section.Addr <= entryPoint && section.Addr+section.Size >= entryPoint {
			offset = getImageBase(f) + section.Offset
			data, err = section.Data()
			if err != nil {
				return nil, err
			}
			break
		}
	}
	if data == nil {
		section := f.Section(".text")
		if section != nil {
			offset = getImageBase(f) + section.Offset
			data, err = section.Data()
			if err != nil {
				return nil, err
			}
		}
	}
	if data == nil {
		for _, segment := range f.Progs {
			if segment.Type == elf.PT_LOAD && segment.Flags == (elf.PF_R&elf.PF_X) {
				if entryPoint > segment.Vaddr {
					segmentData, err := ioutil.ReadAll(segment.Open())
					if err != nil {
						return nil, err
					}
					offset = entryPoint
					if int(entryPoint-segment.Vaddr) > len(segmentData) {
						return nil, errors.New("invalid segment offset")
					}
					data = segmentData[entryPoint-segment.Vaddr:]
					break
				}
			}
		}
	}
	if data != nil {
		engine, err := gapstone.New(arch, mode)
		if err != nil {
			return nil, err
		}
		defer engine.Close()
		instructions, err := engine.Disasm(data, offset, 0)
		if err != nil {
			return nil, err
		}
		symbols := []string{}
		for _, instruction := range instructions {
			if isX86(f) && instruction.Mnemonic == "call" {
				// Consider only call to absolute addresses
				if strings.HasPrefix(instruction.OpStr, "0x") {
					address := instruction.OpStr[2:]
					if !stringMember(symbols, address) {
						symbols = append(symbols, address)
					}
				}
			} else if f.Machine == elf.EM_ARM && strings.HasPrefix(instruction.Mnemonic, "bl") {
				if strings.HasPrefix(instruction.OpStr, "#0x") {
					address := instruction.OpStr[3:]
					if !stringMember(symbols, address) {
						symbols = append(symbols, address)
					}
				}
			} else if f.Machine == elf.EM_MIPS && strings.HasPrefix(instruction.Mnemonic, "lw") {
				if strings.HasPrefix(instruction.OpStr, "$t9, ") {
					address := instruction.OpStr[8 : len(instruction.OpStr)-5]
					if !stringMember(symbols, address) {
						symbols = append(symbols, address)
					}
				}
			}
		}
		return symbols, nil
	}
	return nil, nil
}

func telfhash(elfFile *elf.File) (string, error) {
	symbols := []string{}
	dynSymbols, err := elfFile.DynamicSymbols()
	if err != nil {
		if err != elf.ErrNoSymbols {
			return "", err
		}
	}
	staticSymbols, err := elfFile.Symbols()
	if err != nil {
		if err != elf.ErrNoSymbols {
			return "", err
		}
	}
	if len(staticSymbols) == 0 && len(dynSymbols) == 0 {
		// extract symbols from call sites since we're in a static binary
		symbols, err = extractCallDestinations(elfFile)
		if err != nil {
			return "", err
		}
	} else {
		for _, symbol := range dynSymbols {
			if !canExclude(symbol) {
				symbols = append(symbols, strings.ToLower(symbol.Name))
			}
		}
		for _, symbol := range staticSymbols {
			if !canExclude(symbol) {
				symbols = append(symbols, strings.ToLower(symbol.Name))
			}
		}
		sort.Strings(symbols)
	}
	tlsh := newTlsh()
	tlsh.update([]byte(strings.Join(symbols, ",")))
	return strings.ToLower(tlsh.hash()), nil
}
