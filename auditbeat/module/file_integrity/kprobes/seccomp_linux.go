// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kprobes

import (
	"runtime"

	"github.com/elastic/beats/v7/libbeat/common/seccomp"
)

func init() {
	switch runtime.GOARCH {
	case "amd64", "386", "arm64":
		// The module/file_integrity with kprobes BE uses additional syscalls
		if err := seccomp.ModifyDefaultPolicy(seccomp.AddSyscall,
			"eventfd2",        // required by auditbeat/tracing
			"mount",           // required by auditbeat/tracing
			"perf_event_open", // required by auditbeat/tracing
			"ppoll",           // required by auditbeat/tracing
			"umount2",         // required by auditbeat/tracing
			"truncate",        // required during kprobes verification
			"utime",           // required during kprobes verification
			"utimensat",       // required during kprobes verification
			"setxattr",        // required during kprobes verification
		); err != nil {
			panic(err)
		}
	}
}
