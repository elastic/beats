package main

import (
	"runtime"

	"github.com/elastic/beats/libbeat/common/seccomp"
)

func init() {
	switch runtime.GOARCH {
	case "amd64", "386":
		// The system/package dataset uses librpm which has additional syscall
		// requirements beyond the default policy from libbeat so whitelist
		// these additional syscalls.
		if err := seccomp.ModifyDefaultPolicy(seccomp.AddSyscall, "umask", "mremap"); err != nil {
			panic(err)
		}
	}
}
