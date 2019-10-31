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

package disasm

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/elastic/go-seccomp-bpf/arch"
)

const (
	functionMarker = "TEXT"
)

// Syscall is a system call found in the disassembly.
type Syscall struct {
	Num      int    // Syscall number.
	Name     string // Syscall name.
	Caller   string // Function calling the syscall.
	Function string // Function used to make the systell (e.g. unix.Syscall6).
	Location string // File and line where the syscall is invoked.
	Assembly string // Assembly instruction that loads syscall number.
}

// ExtractSyscalls reads the objdump file and returns the syscalls that it
// finds.
func ExtractSyscalls(arch *arch.Info, objDump string) ([]Syscall, error) {
	var p *parser
	switch arch.ID {
	case i386Parser.ID:
		p = i386Parser
	case x86_64Parser.ID:
		p = x86_64Parser
	default:
		return nil, fmt.Errorf("unsupported architecture %v", arch.Name)
	}

	return p.Parse(objDump)
}

type parser struct {
	*arch.Info
	callOp                 string
	rawSyscallInstructions []string
	parse                  syscallParse
}

type syscallParse func(p *parser, line, caller string, instructions []string) (*Syscall, error)

func (p *parser) Parse(objDump string) ([]Syscall, error) {
	f, err := os.Open(objDump)
	if err != nil {
		return nil, fmt.Errorf("failed to read objdump file: %v", err)
	}
	defer f.Close()

	var function string
	var instructions []string
	var syscalls []Syscall

	s := bufio.NewScanner(bufio.NewReader(f))
	for s.Scan() {
		line := s.Text()
		instructions = append(instructions, line)

		// Find the start of a function.
		if strings.HasPrefix(line, functionMarker) {
			function = line[5:]
			instructions = instructions[:0]
			continue
		}

		syscall, err := p.parse(p, line, function, instructions)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: %v\n", err)
			continue
		}
		if syscall == nil {
			// Line was not a syscall.
			continue
		}

		// Found a syscall. Clear the instruction stack.
		instructions = instructions[:0]

		name, found := p.SyscallNumbers[syscall.Num]
		if !found {
			fmt.Fprintf(os.Stderr, "WARN: unknown syscall %d found at %+v\n", syscall.Num, syscall)
			continue
		}
		syscall.Caller = function
		syscall.Name = name
		syscalls = append(syscalls, *syscall)
	}

	if s.Err() != nil {
		return nil, err
	}

	return syscalls, nil
}

func (p *parser) isRawSyscall(line string) bool {
	for _, ins := range p.rawSyscallInstructions {
		if strings.Contains(line, ins) {
			return true
		}
	}
	return false
}

func (p *parser) isFunctionCall(line string) bool {
	return strings.Contains(line, p.callOp)
}

func isSyscallFunction(function string) bool {
	return strings.Contains(function, "syscall.Syscall(SB)") ||
		strings.Contains(function, "syscall.Syscall6(SB)") ||
		strings.Contains(function, "syscall.rawVforkSyscall(SB)") ||
		strings.Contains(function, "syscall.RawSyscall(SB)") ||
		strings.Contains(function, "syscall.RawSyscall6(SB)") ||
		strings.Contains(function, "unix.RawSyscall(SB)") ||
		strings.Contains(function, "unix.RawSyscall6(SB)") ||
		strings.Contains(function, "unix.RawSyscallNoError(SB)") ||
		strings.Contains(function, "unix.Syscall(SB)") ||
		strings.Contains(function, "unix.Syscall6(SB)") ||
		strings.Contains(function, "unix.Syscall9(SB)") ||
		strings.Contains(function, "unix.SyscallNoError(SB)")
}

func findSyscallNum(instructions []string, syscall *Syscall, matchers ...*regexp.Regexp) error {
	for i := len(instructions) - 1; i >= 0; i-- {
		line := instructions[i]

		for _, regex := range matchers {
			matches := regex.FindStringSubmatch(line)
			if len(matches) != 2 {
				continue
			}

			num, err := strconv.ParseInt(matches[1], 0, 64)
			if err != nil {
				return fmt.Errorf("failed to parse syscall number %v: %v", matches[1], err)
			}
			syscall.Num = int(num)
			syscall.Assembly = matches[0]
			return nil
		}
	}

	return fmt.Errorf("assembly instruction for loading the syscall number was not found")
}

func lastInstruction(instructions []string) string {
	if len(instructions) >= 2 {
		return instructions[len(instructions)-2]
	}
	return ""
}

var (
	x86_64Parser = &parser{
		Info:                   arch.X86_64,
		rawSyscallInstructions: []string{"SYSCALL"},
		callOp:                 "CALL",
		parse:                  parseX86_64,
	}

	i386Parser = &parser{
		Info:                   arch.I386,
		rawSyscallInstructions: []string{"INT $0x80", "SYSENTER"},
		callOp:                 "CALL",
		parse:                  parseX86_64, // i386 is similar to x86_64.
	}
)

var (
	// x86_64SyscallRegex matches the instruction to load the syscall number
	// into a register.
	x86_64SyscallRegex = regexp.MustCompile(`MOV[A-Z]? \$(.+), 0\(SP\)`)

	// x86_64RawSyscallRegex matches the instruction to load the syscall number
	// into a register for a raw "SYSCALL".
	x86_64RawSyscallRegex = regexp.MustCompile(`MOV[A-Z]? \$(.+), (?:AX|BP)`)
)

func parseX86_64(p *parser, line, caller string, instructions []string) (*Syscall, error) {
	var m *regexp.Regexp
	if p.isRawSyscall(line) && !isSyscallFunction(caller) {
		m = x86_64RawSyscallRegex

		// Special case to handle a compiler optimization. This is a read
		// syscall found in cgo binaries.
		if inst := lastInstruction(instructions); inst != "" {
			if strings.Contains(inst, "XORL AX, AX") {
				fields := strings.Fields(line)
				return &Syscall{
					Location: fields[0],
					Function: strings.Join(fields[3:], " "),
					Num:      0,
					Assembly: "XORL AX, AX",
				}, nil
			}
		}
	} else if p.isFunctionCall(line) && isSyscallFunction(line) {
		m = x86_64SyscallRegex
	}

	if m == nil {
		return nil, nil
	}

	fields := strings.Fields(line)
	s := &Syscall{
		Location: fields[0],
		Function: strings.Join(fields[3:], " "),
	}
	if err := findSyscallNum(instructions, s, m); err != nil {
		return nil, fmt.Errorf("failed to extract syscall from '%v': %v",
			strings.TrimSpace(line), err)
	}

	return s, nil
}
