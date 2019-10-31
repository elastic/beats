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

package arch

import (
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

// Info contains Linux architecture information (name, audit arch, and syscall
// tables).
type Info struct {
	Name           string         // Linux architecture name (not necessarily the GOARCH name).
	ID             AuditArch      // Linux audit architecture constant.
	SyscallNames   map[string]int // Mapping of syscall names to numbers.
	SyscallNumbers map[int]string // Mapping of syscall numbers to names.
	SeccompMask    int            // A mask to apply to syscall numbers in BPF instructions (e.g. X32_SYSCALL_BIT).
}

// Linux architecture types.
var (
	ARM = &Info{
		Name:           "arm",
		ID:             auditArchARM,
		SyscallNumbers: syscallsARM,
		SyscallNames:   invert(syscallsARM),
	}
	I386 = &Info{
		Name:           "i386",
		ID:             auditArchI386,
		SyscallNumbers: syscalls386,
		SyscallNames:   invert(syscalls386),
	}
	X32 = &Info{
		// Not a valid GOARCH, but an amd64 binary can use the 32-bit ABI so
		// this can be used to specify those syscalls.
		Name:           "x32",
		ID:             auditArchX86_64,
		SeccompMask:    x32SyscallMask,
		SyscallNumbers: syscallsX32,
		SyscallNames:   invert(syscallsX32),
	}
	X86_64 = &Info{
		Name:           "x86_64",
		ID:             auditArchX86_64,
		SyscallNumbers: syscallsX86_64,
		SyscallNames:   invert(syscallsX86_64),
	}

	// The following architectures are not fully implemented. Syscall tables
	// need to be added for them (syscall number -> name mapping).
	AARCH64 = &Info{
		Name: "aarch64",
		ID:   auditArchAARCH64,
	}
	PPC = &Info{
		Name: "ppc",
		ID:   auditArchPPC,
	}
	PPC64 = &Info{
		Name: "ppc64",
		ID:   auditArchPPC64,
	}
	PPC64LE = &Info{
		Name: "ppc64le",
		ID:   auditArchPPC64LE,
	}
	S390 = &Info{
		Name: "s390",
		ID:   auditArchS390,
	}
	S390X = &Info{
		Name: "s390x",
		ID:   auditArchS390X,
	}
	MIPS = &Info{
		Name: "mips",
		ID:   auditArchMIPS,
	}
	MIPSEL = &Info{
		Name: "mipsel",
		ID:   auditArchMIPSEL,
	}
	MIPS64 = &Info{
		Name: "mips64",
		ID:   auditArchMIPS64,
	}
	MIPS64N32 = &Info{
		Name: "mips64n32",
		ID:   auditArchMIPS64N32,
	}
	MIPSEL64 = &Info{
		Name: "mipsel64",
		ID:   auditArchMIPSEL64,
	}
	MIPSEL64N32 = &Info{
		Name: "mipsel64n32",
		ID:   auditArchMIPSEL64N32,
	}
)

// invert a map[int]string to map[string]int.
func invert(in map[int]string) map[string]int {
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[v] = k
	}
	return out
}

// arches is a mapping of GOARCH and Linux arch names to architecture related
// information.
var arches = map[string]*Info{
	"arm":     ARM,
	"ppc":     PPC,
	"ppc64":   PPC64,
	"ppc64le": PPC64LE,
	"s390":    S390,
	"s390x":   S390X,
	"mips":    MIPS,
	"mipsle":  MIPSEL,
	"mips64":  MIPS64,

	"i386": I386,
	"386":  I386,

	"x32":    X32,
	"x86_64": X86_64,
	"amd64":  X86_64,

	"aarch64": AARCH64,
	"arm64":   AARCH64,

	"mips64n32": MIPS64N32,
	"mips64p32": MIPS64N32,

	"mipsel64": MIPSEL64,
	"mips64le": MIPSEL64,

	"mipsel64n32": MIPSEL64N32,
	"mips64p32le": MIPSEL64N32,
}

// GetInfo returns the arch Info associated with the given architecture name.
// If an architecture is not fully implemented it will return an error.
func GetInfo(name string) (*Info, error) {
	if name == "" {
		name = runtime.GOARCH
	} else {
		name = strings.ToLower(name)
	}

	arch, found := arches[name]
	if !found || len(arch.SyscallNames) == 0 {
		return nil, errors.Errorf("unsupported arch: %v", name)
	}
	return arch, nil
}
