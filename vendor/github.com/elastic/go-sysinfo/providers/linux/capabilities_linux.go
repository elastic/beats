// Copyright 2018 Elasticsearch Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package linux

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
)

func Capabilities() ([]string, error) {
	name := filepath.Join("/proc", strconv.Itoa(os.Getpid()), "status")
	v, err := findValue(name, ":", "CapEff")
	if err != nil {
		return nil, err
	}

	bitmap, err := strconv.ParseUint(v, 16, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse CapEff value")
	}

	var names []string
	for i := 0; i < 64; i++ {
		bit := bitmap & (1 << uint(i))
		if bit > 0 {
			names = append(names, capabilityName(i))
		}
	}

	return names, nil
}

// capabilityNames is mapping of capability constant values to names.
//
// Generated with:
//   curl -s https://raw.githubusercontent.com/torvalds/linux/master/include/uapi/linux/capability.h | \
//   grep -P '^#define CAP_\w+\s+\d+' | perl -pe 's/#define (\w+)\s+(\d+)/\2: "\1",/g'
var capabilityNames = map[int]string{
	0:  "cap_chown",
	1:  "cap_dac_override",
	2:  "cap_dac_read_search",
	3:  "cap_fowner",
	4:  "cap_fsetid",
	5:  "cap_kill",
	6:  "cap_setgid",
	7:  "cap_setuid",
	8:  "cap_setpcap",
	9:  "cap_linux_immutable",
	10: "cap_net_bind_service",
	11: "cap_net_broadcast",
	12: "cap_net_admin",
	13: "cap_net_raw",
	14: "cap_ipc_lock",
	15: "cap_ipc_owner",
	16: "cap_sys_module",
	17: "cap_sys_rawio",
	18: "cap_sys_chroot",
	19: "cap_sys_ptrace",
	20: "cap_sys_pacct",
	21: "cap_sys_admin",
	22: "cap_sys_boot",
	23: "cap_sys_nice",
	24: "cap_sys_resource",
	25: "cap_sys_time",
	26: "cap_sys_tty_config",
	27: "cap_mknod",
	28: "cap_lease",
	29: "cap_audit_write",
	30: "cap_audit_control",
	31: "cap_setfcap",
	32: "cap_mac_override",
	33: "cap_mac_admin",
	34: "cap_syslog",
	35: "cap_wake_alarm",
	36: "cap_block_suspend",
	37: "cap_audit_read",
}

func capabilityName(num int) string {
	name, found := capabilityNames[num]
	if found {
		return name
	}

	return strconv.Itoa(num)
}

func findValue(filename, separator, key string) (string, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}

	var line []byte
	sc := bufio.NewScanner(bytes.NewReader(content))
	for sc.Scan() {
		if bytes.HasPrefix(sc.Bytes(), []byte(key)) {
			line = sc.Bytes()
			break
		}
	}
	if len(line) == 0 {
		return "", errors.Errorf("%v not found", key)
	}

	parts := bytes.SplitN(line, []byte(separator), 2)
	if len(parts) != 2 {
		return "", errors.Errorf("unexpected line format for '%v'", string(line))
	}

	return string(bytes.TrimSpace(parts[1])), nil
}
