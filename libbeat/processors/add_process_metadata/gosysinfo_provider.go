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

package add_process_metadata

import (
	"os/user"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common/capabilities"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

type gosysinfoProvider struct{}

func (p gosysinfoProvider) GetProcessMetadata(pid int) (result *processMetadata, err error) {
	proc, err := sysinfo.Process(pid)
	if err != nil {
		return nil, err
	}

	var info types.ProcessInfo
	info, err = proc.Info()
	if err != nil {
		return nil, err
	}

	var env map[string]string
	if e, ok := proc.(types.Environment); ok {
		env, _ = e.Environment()
	}

	username, userid := "", ""
	if userInfo, err := proc.User(); err == nil {
		userid = userInfo.UID
		if u, err := user.LookupId(userInfo.UID); err == nil {
			username = u.Username
		}
	}

	capPermitted, _ := capabilities.FromPid(capabilities.Permitted, pid)
	capEffective, _ := capabilities.FromPid(capabilities.Effective, pid)

	r := processMetadata{
		name:      info.Name,
		args:      info.Args,
		env:       env,
		title:     strings.Join(info.Args, " "),
		exe:       info.Exe,
		pid:       info.PID,
		ppid:      info.PPID,
		capEffective: capEffective,
		capPermitted: capPermitted,
		startTime: info.StartTime,
		username:  username,
		userid:    userid,
	}
	r.fields = r.toMap()
	return &r, nil
}
