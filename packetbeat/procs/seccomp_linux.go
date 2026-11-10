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

//go:build linux && amd64 && cgo

package procs

import (
	"github.com/elastic/beats/v7/libbeat/common/seccomp"
)

func init() {
	// The default seccomp policy on amd64 is an allowlist that predates
	// eBPF. Allow the system calls quark makes when opening a queue and
	// loading its programs, otherwise the kernel_tracing backend fails
	// with EPERM.
	//
	// This must stay amd64-only: ModifyDefaultPolicy appends to the
	// first syscall group of the default policy, which on arm64 is an
	// errno denylist, so applying it there would block these calls
	// instead of allowing them. The arm64 default already allows them.
	if err := seccomp.ModifyDefaultPolicy(seccomp.AddSyscall,
		"bpf",             // load programs and maps
		"perf_event_open", // attach probes
		"memfd_create",    // libbpf object loading
	); err != nil {
		panic(err)
	}
}
