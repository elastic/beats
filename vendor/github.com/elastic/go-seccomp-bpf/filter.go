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

package seccomp

import (
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/bpf"

	"github.com/elastic/go-seccomp-bpf/arch"
)

const (
	syscallNumOffset = 0
	archOffset       = 4
)

// FilterFlag is a flag that is passed to the seccomp. Multiple flags can be
// OR'ed together.
type FilterFlag uint32

var filterFlagNames = map[FilterFlag]string{
	FilterFlagTSync: "tsync",
	FilterFlagLog:   "log",
}

// String returns a string representation of the FilterFlag.
func (f FilterFlag) String() string {
	if name, found := filterFlagNames[f]; found {
		return name
	}

	var list []string
	for flag, name := range filterFlagNames {
		if f&flag != 0 {
			f ^= flag
			list = append(list, name)
		}
	}
	if f != 0 {
		list = append(list, "unknown")
	}
	return strings.Join(list, "|")
}

// MarshalText marshals the value to text.
func (f FilterFlag) MarshalText() ([]byte, error) {
	return []byte(f.String()), nil
}

// Action specifies what to do when a syscall matches during filter evaluation.
type Action uint32

var actionNames = map[Action]string{
	ActionKillThread:  "kill_thread",
	ActionKillProcess: "kill_process",
	ActionTrap:        "trap",
	ActionErrno:       "errno",
	ActionTrace:       "trace",
	ActionLog:         "log",
	ActionAllow:       "allow",
}

// Unpack sets the Action value based on the string.
func (a *Action) Unpack(s string) error {
	s = strings.ToLower(s)
	for action, name := range actionNames {
		if name == s {
			*a = action
			return nil
		}
	}
	return errors.Errorf("invalid action: %v", s)
}

// String returns a string representation of the Action.
func (a Action) String() string {
	name, found := actionNames[a]
	if found {
		return name
	}
	return "unknown"
}

// MarshalText marshals the value to text.
func (a Action) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}

// Filter contains all the parameters necessary to install a Linux seccomp
// filter for the process.
type Filter struct {
	NoNewPrivs bool       `config:"no_new_privs" json:"no_new_privs"` // Set the process's no new privs bit.
	Flag       FilterFlag `config:"flag"         json:"flag"`         // Flag to pass to the seccomp call.
	Policy     Policy     `config:"policy"       json:"policy"`       // Policy that will be assembled into a BPF filter.
}

// Policy defines the BPF seccomp filter.
type Policy struct {
	DefaultAction Action         `config:"default_action" json:"default_action" yaml:"default_action"` // Action when no syscalls match.
	Syscalls      []SyscallGroup `config:"syscalls"       json:"syscalls"       yaml:"syscalls"`       // Groups of syscalls and actions.

	arch *arch.Info
}

// SyscallGroup is a logical block within a Policy that contains a set of
// syscalls to match against and an action to take.
type SyscallGroup struct {
	Names  []string `config:"names"  validate:"required" json:"names"  yaml:"names"`  // List of syscall names (all must exist).
	Action Action   `config:"action" validate:"required" json:"action" yaml:"action"` // Action to take upon a match.

	arch *arch.Info
}

// Validate validates that the configuration has both a default action and a
// set of syscalls.
func (p *Policy) Validate() error {
	if _, found := actionNames[p.DefaultAction]; !found {
		return errors.Errorf("invalid default_action value %d", p.DefaultAction)
	}

	if len(p.Syscalls) == 0 {
		return errors.New("syscalls must not be empty")
	}

	return nil
}

// Assemble assembles the policy into a list of BPF instructions. If the policy
// contains any unknown syscalls or invalid actions an error will be returned.
func (p *Policy) Assemble() ([]bpf.Instruction, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	// Ensure arch has been set for the policy.
	if p.arch == nil {
		arch, err := arch.GetInfo("")
		if err != nil {
			return nil, err
		}
		p.arch = arch
	}

	// Setup the action.
	action := p.DefaultAction
	if action == ActionErrno {
		action |= Action(errnoEPERM)
	}

	// Build the syscall filters.
	var instructions []bpf.Instruction
	for _, group := range p.Syscalls {
		if group.arch == nil {
			group.arch = p.arch
		}

		groupInsts, err := group.Assemble()
		if err != nil {
			return nil, err
		}

		instructions = append(instructions, groupInsts...)
	}

	// Filter out x32 to prevent bypassing blacklists by using the 32-bit ABI.
	var x32Filter []bpf.Instruction
	if p.arch.ID == arch.X86_64.ID {
		x32Filter = []bpf.Instruction{
			bpf.JumpIf{Cond: bpf.JumpGreaterOrEqual, Val: uint32(arch.X32.SeccompMask), SkipFalse: 1},
			bpf.RetConstant{Val: uint32(ActionErrno) | uint32(errnoENOSYS)},
		}
	}

	program := make([]bpf.Instruction, 0, len(x32Filter)+len(instructions)+5)

	program = append(program, bpf.LoadAbsolute{Off: archOffset, Size: 4})

	// If the loaded arch ID is not equal p.arch.ID, jump to the final Ret instruction.
	jumpN := 1 + len(x32Filter) + len(instructions)
	if jumpN <= 255 {
		program = append(program, bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: uint32(p.arch.ID), SkipTrue: uint8(jumpN)})
	} else {
		// JumpIf can not handle long jumps, so we switch to two instructions for this case.
		program = append(program, bpf.JumpIf{Cond: bpf.JumpEqual, Val: uint32(p.arch.ID), SkipTrue: 1})
		program = append(program, bpf.Jump{Skip: uint32(jumpN)})
	}

	program = append(program, bpf.LoadAbsolute{Off: syscallNumOffset, Size: 4})
	program = append(program, x32Filter...)
	program = append(program, instructions...)
	program = append(program, bpf.RetConstant{Val: uint32(action)})
	return program, nil
}

// Dump writes a textual represenation of the BPF instructions to out.
func (p *Policy) Dump(out io.Writer) error {
	assembled, err := p.Assemble()
	if err != nil {
		return err
	}

	for n, instruction := range assembled {
		fmt.Fprintf(out, "%d: %v\n", n, instruction)
	}
	return nil
}

// Assemble assembles the policy into a list of BPF instructions. If the group
// contains any unknown syscalls or invalid actions an error will be returned.
func (g *SyscallGroup) Assemble() ([]bpf.Instruction, error) {
	if len(g.Names) == 0 {
		return nil, nil
	}

	// Validate the syscalls.
	var unknowns []string
	var syscallNums []uint32
	for _, name := range g.Names {
		if num, found := g.arch.SyscallNames[name]; found {
			syscallNums = append(syscallNums, uint32(num|g.arch.SeccompMask))
		} else {
			unknowns = append(unknowns, name)
		}
	}
	if len(unknowns) > 0 {
		return nil, errors.Errorf("found unknown syscalls for arch %v: %v",
			g.arch.Name, strings.Join(unknowns, ","))
	}

	insts := make([]bpf.Instruction, 0, len(syscallNums))

	for len(syscallNums) > 0 {
		// JumpIf can not handle long jumps, so we insert a Ret after each group of 256 checks.
		size := len(syscallNums)
		if size > 256 {
			size = 256
		}
		syscallNums256 := syscallNums[:size]
		syscallNums = syscallNums[size:]

		for i, num := range syscallNums256 {
			jumpN := uint8(len(syscallNums256) - i - 1)
			jeq := bpf.JumpIf{Cond: bpf.JumpEqual, Val: num, SkipTrue: jumpN}
			if jumpN == 0 {
				// Skip the return statement if the last condition was not satisfied.
				jeq.SkipFalse = 1
			}
			insts = append(insts, jeq)
		}

		action := g.Action
		if action == ActionErrno {
			action |= Action(errnoEPERM)
		}
		insts = append(insts, bpf.RetConstant{Val: uint32(action)})
	}

	return insts, nil
}
