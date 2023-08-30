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
	// All test cases were constructed based on behaviour of capsh --decode <journald.process.capabilities>.
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
							"cap_chown",
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
							"cap_chown",
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
							"cap_chown",
							"cap_dac_override",
							"cap_dac_read_search",
							"cap_fowner",
							"cap_fsetid",
							"cap_kill",
							"cap_setgid",
							"cap_setuid",
							"cap_setpcap",
							"cap_linux_immutable",
							"cap_net_bind_service",
							"cap_net_broadcast",
							"cap_net_admin",
							"cap_net_raw",
							"cap_ipc_lock",
							"cap_ipc_owner",
							"cap_sys_module",
							"cap_sys_rawio",
							"cap_sys_chroot",
							"cap_sys_ptrace",
							"cap_sys_pacct",
							"cap_sys_admin",
							"cap_sys_boot",
							"cap_sys_nice",
							"cap_sys_resource",
							"cap_sys_time",
							"cap_sys_tty_config",
							"cap_mknod",
							"cap_lease",
							"cap_audit_write",
							"cap_audit_control",
							"cap_setfcap",
							"cap_mac_override",
							"cap_mac_admin",
							"cap_syslog",
							"cap_wake_alarm",
							"cap_block_suspend",
							"cap_audit_read",
							"cap_perfmon",
							"cap_bpf",
							"cap_checkpoint_restore",
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
							"cap_chown",
							"cap_dac_override",
							"cap_dac_read_search",
							"cap_fowner",
							"cap_fsetid",
							"cap_kill",
							"cap_setgid",
							"cap_setuid",
							"cap_setpcap",
							"cap_linux_immutable",
							"cap_net_bind_service",
							"cap_net_broadcast",
							"cap_net_admin",
							"cap_net_raw",
							"cap_ipc_lock",
							"cap_ipc_owner",
							"cap_sys_module",
							"cap_sys_rawio",
							"cap_sys_chroot",
							"cap_sys_ptrace",
							"cap_sys_pacct",
							"cap_sys_admin",
							"cap_sys_boot",
							"cap_sys_nice",
							"cap_sys_resource",
							"cap_sys_time",
							"cap_sys_tty_config",
							"cap_mknod",
							"cap_lease",
							"cap_audit_write",
							"cap_audit_control",
							"cap_setfcap",
							"cap_mac_override",
							"cap_mac_admin",
							"cap_syslog",
							"cap_wake_alarm",
							"cap_block_suspend",
							"cap_audit_read",
							"cap_perfmon",
							"cap_bpf",
							"cap_checkpoint_restore",
							"41",
							"42",
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
							"cap_chown",
							"cap_dac_override",
							"cap_dac_read_search",
							"cap_fowner",
							"cap_kill",
							"cap_setgid",
							"cap_setuid",
							"cap_linux_immutable",
							"cap_net_bind_service",
							"cap_net_broadcast",
							"cap_net_admin",
							"cap_net_raw",
							"cap_ipc_owner",
							"cap_sys_module",
							"cap_sys_chroot",
							"cap_sys_ptrace",
							"cap_sys_admin",
							"cap_sys_nice",
							"cap_sys_time",
							"cap_sys_tty_config",
							"cap_mknod",
							"cap_lease",
							"cap_audit_control",
							"cap_setfcap",
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
