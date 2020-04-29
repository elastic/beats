// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"runtime"

	"github.com/elastic/beats/v7/libbeat/common/seccomp"
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

		// The system/socket dataset uses additional syscalls
		if err := seccomp.ModifyDefaultPolicy(seccomp.AddSyscall, "perf_event_open", "eventfd2", "ppoll", "mount", "umount2"); err != nil {
			panic(err)
		}
	}
}
