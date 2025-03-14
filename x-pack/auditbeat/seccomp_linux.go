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
		if err := seccomp.ModifyDefaultPolicy(seccomp.AddSyscall,
			"mremap",
			"umask",
			"setreuid",
		); err != nil {
			panic(err)
		}

		// The system/socket dataset uses additional syscalls
		if err := seccomp.ModifyDefaultPolicy(seccomp.AddSyscall,
			"eventfd2",
			"mount",
			"mq_open", // required for creds kprobe guess trigger.
			"perf_event_open",
			"ppoll",
			"umount2",
		); err != nil {
			panic(err)
		}

		// The sessionmd processor kerneltracingprovider needs
		// memfd_create to operate via EBPF
		if err := seccomp.ModifyDefaultPolicy(seccomp.AddSyscall,
			"memfd_create",
		); err != nil {
			panic(err)
		}

		// The system/process dataset uses additional syscalls
		if err := seccomp.ModifyDefaultPolicy(seccomp.AddSyscall,
			"statx",
		); err != nil {
			panic(err)
		}
	}
}
