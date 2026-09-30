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

//go:build linux && (amd64 || arm64) && cgo

package file_integrity

import (
	"runtime"

	"github.com/elastic/beats/v7/libbeat/common/seccomp"

	"github.com/elastic/elastic-agent-libs/logp"
)

func init() {
	// quark's eBPF backend loads programs and reads perf ring buffers, which
	// the default seccomp policy does not allow. This package is linked into
	// both the OSS and x-pack auditbeat binaries, so the x-pack quark
	// consumers rely on this policy extension as well.
	if runtime.GOARCH == "amd64" {
		if err := seccomp.ModifyDefaultPolicy(seccomp.AddSyscall,
			"bpf",
			"eventfd2",        // needed by ring buffers
			"memfd_create",    // needed when quark loads its embedded BTF
			"perf_event_open", // needed by tracepoints and kprobes
		); err != nil {
			panic(err)
		}
	}
}

func newEBPFReader(c Config, l *logp.Logger) (EventProducer, error) {
	paths := make(map[string]struct{})
	for _, p := range c.Paths {
		paths[p] = struct{}{}
	}

	return &ebpfReader{
		config:  c,
		log:     l,
		parsers: FileParsers(c),
		paths:   paths,
		eventC:  make(chan Event),
	}, nil
}
