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

//go:build linux

package sys

import (
	"encoding/binary"
	"sync"
	"time"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

var (
	hostIDOnce struct {
		sync.Once
		value string
		err   error
	}
)

func HostID() (string, error) {
	hostIDOnce.Do(func() {
		var info types.Host
		info, hostIDOnce.err = sysinfo.Host()
		if hostIDOnce.err != nil {
			return
		}
		hostIDOnce.value = info.Info().UniqueID

	})
	return hostIDOnce.value, hostIDOnce.err
}

// EntityID creates an ID that uniquely identifies this process across machines.
func EntityID(pid uint32, start time.Time) (string, error) {
	hid, err := HostID()
	if err != nil {
		return "", err
	}

	h := system.NewEntityHash()
	h.Write([]byte(hid))
	if err := binary.Write(h, binary.LittleEndian, int64(pid)); err != nil {
		return "", err
	}
	if err := binary.Write(h, binary.LittleEndian, int64(start.Nanosecond())); err != nil {
		return "", err
	}

	return h.Sum(), nil
}
