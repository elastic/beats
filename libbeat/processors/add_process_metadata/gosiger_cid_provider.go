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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/elastic/gosigar/cgroup"
)

type gosigarCidProvider struct {
	hostPath     string
	cgroupPrefix string
}

func (p gosigarCidProvider) GetCid(pid int) (result string, err error) {
	cgroups, err := cgroup.ProcessCgroupPaths(p.hostPath, pid)
	if err != nil {
		return "", fmt.Errorf("failed to read cgroups for pid=%v", pid)
	}
	cid := p.getCid(cgroups)
	return cid, nil
}

func newCidProvider(hostPath string, cgroupPrefix string) gosigarCidProvider {
	return gosigarCidProvider{
		hostPath:     hostPath,
		cgroupPrefix: cgroupPrefix,
	}
}

// getCid checks all of the processes' paths to see if any
// of them are associated with Kubernetes. Kubernetes uses /kubepods/<quality>/<podId>/<cid> when
// naming cgroups and we use this to determine the container ID. If no container
// ID is found then an empty string is returned.
// Example:
// /kubepods/besteffort/pod9b9e44c2-00fd-11ea-95e9-080027421ddf/2bb9fd4de339e5d4f094e78bb87636004acfe53f5668104addc761fe4a93588e
func (p gosigarCidProvider) getCid(cgroups map[string]string) string {
	for _, path := range cgroups {
		if strings.HasPrefix(path, p.cgroupPrefix) {
			return filepath.Base(path)
		}
	}
	return ""
}
