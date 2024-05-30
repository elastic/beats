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
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"os/user"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/capabilities"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

var hostInfoOnce = sync.OnceValues(func() ([]byte, error) {
	host, err := sysinfo.Host()
	if err == nil {
		if uniqueID := host.Info().UniqueID; uniqueID != "" {
			return []byte(uniqueID), err
		}
	}

	return nil, err
})

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

	username, userid, groupname, groupid := "", "", "", ""
	if userInfo, err := proc.User(); err == nil {
		userid = userInfo.UID
		if u, err := user.LookupId(userInfo.UID); err == nil {
			username = u.Username
		}

		groupid = userInfo.GID
		if g, err := user.LookupGroupId(userInfo.GID); err == nil {
			groupname = g.Name
		}
	}

	eID, _ := entityID(pid, info.StartTime)

	// Capabilities are linux only and other systems will fail
	// with ErrUnsupported. In the event of any errors, we simply
	// don't report the capabilities.
	capPermitted, _ := capabilities.FromPid(capabilities.Permitted, pid)
	capEffective, _ := capabilities.FromPid(capabilities.Effective, pid)

	r := processMetadata{
		entityID:     eID,
		name:         info.Name,
		args:         info.Args,
		env:          env,
		title:        strings.Join(info.Args, " "),
		exe:          info.Exe,
		pid:          info.PID,
		ppid:         info.PPID,
		capEffective: capEffective,
		capPermitted: capPermitted,
		startTime:    info.StartTime,
		username:     username,
		userid:       userid,
		groupname:    groupname,
		groupid:      groupid,
	}

	r.fields = r.toMap()
	return &r, nil
}

// entityID creates an ID that uniquely identifies this process across machines.
func entityID(pid int, start time.Time) (string, error) {
	uniqueID, err := hostInfoOnce()
	if err != nil && len(uniqueID) == 0 {
		return "", err
	}

	if len(uniqueID) == 0 || start.IsZero() {
		return "", nil
	}

	h := sha256.New()
	if _, err := h.Write(uniqueID); err != nil {
		return "", err
	}
	if err := binary.Write(h, binary.LittleEndian, int64(pid)); err != nil {
		return "", err
	}
	if err := binary.Write(h, binary.LittleEndian, int64(start.Nanosecond())); err != nil {
		return "", err
	}

	sum := h.Sum(nil)
	if len(sum) > 12 {
		sum = sum[:12]
	}
	return base64.RawStdEncoding.EncodeToString(sum), nil
}
