// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package ebpf

import (
	"runtime"

	"github.com/elastic/beats/v7/libbeat/common/seccomp"
)

func init() {
	switch runtime.GOARCH {
	case "amd64", "arm64":
		syscalls := []string{
			"bpf",
			"eventfd2",        // needed by ringbuf
			"perf_event_open", // needed by tracepoints
			"openat",          // needed to create map
			"newfstatat",      // needed for BTF
		}
		if err := seccomp.ModifyDefaultPolicy(seccomp.AddSyscall, syscalls...); err != nil {
			panic(err)
		}
	}
}
