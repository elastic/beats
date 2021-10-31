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

package linux

import (
	"strconv"

	"github.com/elastic/go-sysinfo/types"
)

// capabilityNames is mapping of capability constant values to names.
//
// Generated with:
//   curl -s https://raw.githubusercontent.com/torvalds/linux/master/include/uapi/linux/capability.h | \
//   grep -P '^#define CAP_\w+\s+\d+' | perl -pe 's/#define (\w+)\s+(\d+)/\2: "\1",/g'
var capabilityNames = map[int]string{
	0:  "chown",
	1:  "dac_override",
	2:  "dac_read_search",
	3:  "fowner",
	4:  "fsetid",
	5:  "kill",
	6:  "setgid",
	7:  "setuid",
	8:  "setpcap",
	9:  "linux_immutable",
	10: "net_bind_service",
	11: "net_broadcast",
	12: "net_admin",
	13: "net_raw",
	14: "ipc_lock",
	15: "ipc_owner",
	16: "sys_module",
	17: "sys_rawio",
	18: "sys_chroot",
	19: "sys_ptrace",
	20: "sys_pacct",
	21: "sys_admin",
	22: "sys_boot",
	23: "sys_nice",
	24: "sys_resource",
	25: "sys_time",
	26: "sys_tty_config",
	27: "mknod",
	28: "lease",
	29: "audit_write",
	30: "audit_control",
	31: "setfcap",
	32: "mac_override",
	33: "mac_admin",
	34: "syslog",
	35: "wake_alarm",
	36: "block_suspend",
	37: "audit_read",
}

func capabilityName(num int) string {
	name, found := capabilityNames[num]
	if found {
		return name
	}

	return strconv.Itoa(num)
}

func readCapabilities(content []byte) (*types.CapabilityInfo, error) {
	var cap types.CapabilityInfo

	err := parseKeyValue(content, ":", func(key, value []byte) error {
		var err error
		switch string(key) {
		case "CapInh":
			cap.Inheritable, err = decodeBitMap(string(value), capabilityName)
			if err != nil {
				return err
			}
		case "CapPrm":
			cap.Permitted, err = decodeBitMap(string(value), capabilityName)
			if err != nil {
				return err
			}
		case "CapEff":
			cap.Effective, err = decodeBitMap(string(value), capabilityName)
			if err != nil {
				return err
			}
		case "CapBnd":
			cap.Bounding, err = decodeBitMap(string(value), capabilityName)
			if err != nil {
				return err
			}
		case "CapAmb":
			cap.Ambient, err = decodeBitMap(string(value), capabilityName)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return &cap, err
}
