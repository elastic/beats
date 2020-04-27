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
		// The linux/perf dataset uses additional syscalls
		if err := seccomp.ModifyDefaultPolicy(seccomp.AddSyscall, "perf_event_open"); err != nil {
			panic(err)
		}
	}
}
