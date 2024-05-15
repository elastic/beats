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
