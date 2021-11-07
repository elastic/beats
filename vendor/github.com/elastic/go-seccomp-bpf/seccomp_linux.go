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
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
	"golang.org/x/net/bpf"
	"golang.org/x/sys/unix"
)

// Supported returns true if the seccomp syscall is supported.
func Supported() bool {
	// Strict mode requires that flags be set to 0, but we are sending 1 so
	// this will return EINVAL if the syscall exists and is allowed.
	if err := seccomp(seccompSetModeStrict, 1, nil); err == syscall.EINVAL {
		return true
	}

	return false
}

// SetNoNewPrivs will use prctl to set the calling thread's no_new_privs bit to
// 1 (true). Once set, this bit cannot be unset.
func SetNoNewPrivs() error {
	return prctl(prSetNoNewPrivs, 1)
}

// LoadFilter will install seccomp using native methods.
func LoadFilter(filter Filter) error {
	insts, err := filter.Policy.Assemble()
	if err != nil {
		return errors.Wrap(err, "failed to assemble policy")
	}

	raw, err := bpf.Assemble(insts)
	if err != nil {
		return errors.Wrap(err, "failed to assemble BPF instructions")
	}

	sockFilter := sockFilter(raw)
	program := &syscall.SockFprog{
		Len:    uint16(len(sockFilter)),
		Filter: &sockFilter[0],
	}

	if filter.NoNewPrivs {
		if err = SetNoNewPrivs(); err != nil {
			return errors.Wrap(err, "failed to set no_new_privs with prctl")
		}
	}

	if err = seccomp(seccompSetModeFilter, filter.Flag, unsafe.Pointer(program)); err != nil {
		if err == syscall.ENOSYS {
			return errors.Wrap(err, "failed loading seccomp filter: seccomp "+
				"is not supported by the kernel")
		}
		return errors.Wrap(err, "failed loading seccomp filter")
	}

	return nil
}

func sockFilter(raw []bpf.RawInstruction) []syscall.SockFilter {
	filter := make([]syscall.SockFilter, 0, len(raw))
	for _, instruction := range raw {
		filter = append(filter, syscall.SockFilter{
			Code: instruction.Op,
			Jt:   instruction.Jt,
			Jf:   instruction.Jf,
			K:    instruction.K,
		})
	}
	return filter
}

// prctl syscall wrapper.
func prctl(option uintptr, args ...uintptr) error {
	if len(args) > 4 {
		return syscall.E2BIG
	}
	var arg [4]uintptr
	copy(arg[:], args)
	_, _, e := syscall.Syscall6(syscall.SYS_PRCTL, option, arg[0], arg[1], arg[2], arg[3], 0)
	if e != 0 {
		return e
	}
	return nil
}

// seccomp syscall wrapper.
func seccomp(op uintptr, flags FilterFlag, uargs unsafe.Pointer) error {
	_, _, e := syscall.Syscall(unix.SYS_SECCOMP, op, uintptr(flags), uintptr(uargs))
	if e != 0 {
		return e
	}
	return nil
}
