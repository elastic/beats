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

//go:build linux && cgo

package journalfield

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

var expandCapabilitiesTests = []struct {
	name string
	src  mapstr.M
	want mapstr.M
}{
	// All test cases were constructed based on behaviour of capsh --decode <journald.process.capabilities>,
	// with the exception that the CONSTANT names are used instead of the canonical lowercase names in order
	// to conform with ECS directions.
	{
		name: "none",
		src: mapstr.M{
			"journald": mapstr.M{
				"process": mapstr.M{
					"capabilities": "0",
				},
			},
		},
		want: mapstr.M{
			"journald": mapstr.M{
				"process": mapstr.M{
					"capabilities": "0",
				},
			},
		},
	},
	{
		name: "cap_chown_short",
		src: mapstr.M{
			"journald": mapstr.M{
				"process": mapstr.M{
					"capabilities": "1",
				},
			},
		},
		want: mapstr.M{
			"journald": mapstr.M{
				"process": mapstr.M{
					"capabilities": "1",
				},
			},
			"process": mapstr.M{
				"thread": mapstr.M{
					"capabilities": mapstr.M{
						"effective": []string{
							"CAP_CHOWN",
						},
					},
				},
			},
		},
	},
	{
		name: "cap_chown_long",
		src: mapstr.M{
			"journald": mapstr.M{
				"process": mapstr.M{
					"capabilities": "0000000000000001",
				},
			},
		},
		want: mapstr.M{
			"journald": mapstr.M{
				"process": mapstr.M{
					"capabilities": "0000000000000001",
				},
			},
			"process": mapstr.M{
				"thread": mapstr.M{
					"capabilities": mapstr.M{
						"effective": []string{
							"CAP_CHOWN",
						},
					},
				},
			},
		},
	},
	{
		name: "all",
		src: mapstr.M{
			"journald": mapstr.M{
				"process": mapstr.M{
					"capabilities": "1ffffffffff",
				},
			},
		},
		want: mapstr.M{
			"journald": mapstr.M{
				"process": mapstr.M{
					"capabilities": "1ffffffffff",
				},
			},
			"process": mapstr.M{
				"thread": mapstr.M{
					"capabilities": mapstr.M{
						"effective": []string{
							"CAP_CHOWN",
							"CAP_DAC_OVERRIDE",
							"CAP_DAC_READ_SEARCH",
							"CAP_FOWNER",
							"CAP_FSETID",
							"CAP_KILL",
							"CAP_SETGID",
							"CAP_SETUID",
							"CAP_SETPCAP",
							"CAP_LINUX_IMMUTABLE",
							"CAP_NET_BIND_SERVICE",
							"CAP_NET_BROADCAST",
							"CAP_NET_ADMIN",
							"CAP_NET_RAW",
							"CAP_IPC_LOCK",
							"CAP_IPC_OWNER",
							"CAP_SYS_MODULE",
							"CAP_SYS_RAWIO",
							"CAP_SYS_CHROOT",
							"CAP_SYS_PTRACE",
							"CAP_SYS_PACCT",
							"CAP_SYS_ADMIN",
							"CAP_SYS_BOOT",
							"CAP_SYS_NICE",
							"CAP_SYS_RESOURCE",
							"CAP_SYS_TIME",
							"CAP_SYS_TTY_CONFIG",
							"CAP_MKNOD",
							"CAP_LEASE",
							"CAP_AUDIT_WRITE",
							"CAP_AUDIT_CONTROL",
							"CAP_SETFCAP",
							"CAP_MAC_OVERRIDE",
							"CAP_MAC_ADMIN",
							"CAP_SYSLOG",
							"CAP_WAKE_ALARM",
							"CAP_BLOCK_SUSPEND",
							"CAP_AUDIT_READ",
							"CAP_PERFMON",
							"CAP_BPF",
							"CAP_CHECKPOINT_RESTORE",
						},
					},
				},
			},
		},
	},
	{
		name: "all_and_new",
		src: mapstr.M{
			"journald": mapstr.M{
				"process": mapstr.M{
					"capabilities": "7ffffffffff",
				},
			},
		},
		want: mapstr.M{
			"journald": mapstr.M{
				"process": mapstr.M{
					"capabilities": "7ffffffffff",
				},
			},
			"process": mapstr.M{
				"thread": mapstr.M{
					"capabilities": mapstr.M{
						"effective": []string{
							"CAP_CHOWN",
							"CAP_DAC_OVERRIDE",
							"CAP_DAC_READ_SEARCH",
							"CAP_FOWNER",
							"CAP_FSETID",
							"CAP_KILL",
							"CAP_SETGID",
							"CAP_SETUID",
							"CAP_SETPCAP",
							"CAP_LINUX_IMMUTABLE",
							"CAP_NET_BIND_SERVICE",
							"CAP_NET_BROADCAST",
							"CAP_NET_ADMIN",
							"CAP_NET_RAW",
							"CAP_IPC_LOCK",
							"CAP_IPC_OWNER",
							"CAP_SYS_MODULE",
							"CAP_SYS_RAWIO",
							"CAP_SYS_CHROOT",
							"CAP_SYS_PTRACE",
							"CAP_SYS_PACCT",
							"CAP_SYS_ADMIN",
							"CAP_SYS_BOOT",
							"CAP_SYS_NICE",
							"CAP_SYS_RESOURCE",
							"CAP_SYS_TIME",
							"CAP_SYS_TTY_CONFIG",
							"CAP_MKNOD",
							"CAP_LEASE",
							"CAP_AUDIT_WRITE",
							"CAP_AUDIT_CONTROL",
							"CAP_SETFCAP",
							"CAP_MAC_OVERRIDE",
							"CAP_MAC_ADMIN",
							"CAP_SYSLOG",
							"CAP_WAKE_ALARM",
							"CAP_BLOCK_SUSPEND",
							"CAP_AUDIT_READ",
							"CAP_PERFMON",
							"CAP_BPF",
							"CAP_CHECKPOINT_RESTORE",
							"CAP_41",
							"CAP_42",
						},
					},
				},
			},
		},
	},
	{
		name: "sparse",
		src: mapstr.M{
			"journald": mapstr.M{
				"process": mapstr.M{
					"capabilities": "deadbeef",
				},
			},
		},
		want: mapstr.M{
			"journald": mapstr.M{
				"process": mapstr.M{
					"capabilities": "deadbeef",
				},
			},
			"process": mapstr.M{
				"thread": mapstr.M{
					"capabilities": mapstr.M{
						"effective": []string{
							"CAP_CHOWN",
							"CAP_DAC_OVERRIDE",
							"CAP_DAC_READ_SEARCH",
							"CAP_FOWNER",
							"CAP_KILL",
							"CAP_SETGID",
							"CAP_SETUID",
							"CAP_LINUX_IMMUTABLE",
							"CAP_NET_BIND_SERVICE",
							"CAP_NET_BROADCAST",
							"CAP_NET_ADMIN",
							"CAP_NET_RAW",
							"CAP_IPC_OWNER",
							"CAP_SYS_MODULE",
							"CAP_SYS_CHROOT",
							"CAP_SYS_PTRACE",
							"CAP_SYS_ADMIN",
							"CAP_SYS_NICE",
							"CAP_SYS_TIME",
							"CAP_SYS_TTY_CONFIG",
							"CAP_MKNOD",
							"CAP_LEASE",
							"CAP_AUDIT_CONTROL",
							"CAP_SETFCAP",
						},
					},
				},
			},
		},
	},
}

func TestExpandCapabilities(t *testing.T) {
	for _, test := range expandCapabilitiesTests {
		t.Run(test.name, func(t *testing.T) {
			dst := test.src.Clone()
			expandCapabilities(dst)
			got := dst
			if !cmp.Equal(test.want, got) {
				t.Errorf("unexpected result\n--- want\n+++ got\n%s", cmp.Diff(test.want, got))
			}
		})
	}
}
