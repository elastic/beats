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
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/libbeat/metric/system/cgroup"
	"github.com/elastic/beats/v8/libbeat/metric/system/resolve"
)

const (
	providerName = "gosigar_cid_provider"
)

type gosigarCidProvider struct {
	log                *logp.Logger
	hostPath           resolve.Resolver
	cgroupPrefixes     []string
	cgroupRegex        string
	cidRegex           *regexp.Regexp
	processCgroupPaths func(resolve.Resolver, int) (cgroup.PathList, error)
	pidCidCache        *common.Cache
}

func (p gosigarCidProvider) GetCid(pid int) (result string, err error) {

	var cid string
	var ok bool

	// check from cache
	if p.pidCidCache != nil {
		if cid, ok = p.pidCidCache.Get(pid).(string); ok {
			p.log.Debugf("Using cached container id for pid=%v", pid)
			return cid, nil
		}
	}

	cgroups, err := p.getProcessCgroups(pid)

	if err != nil {
		p.log.Debugf("failed to get cgroups for pid=%v: %v", pid, err)
	}

	cid = p.getCid(cgroups)

	// add pid and cid to cache
	if p.pidCidCache != nil {
		p.pidCidCache.Put(pid, cid)
	}
	return cid, nil
}

func newCidProvider(hostPath resolve.Resolver, cgroupPrefixes []string, cgroupRegex string, processCgroupPaths func(resolve.Resolver, int) (cgroup.PathList, error), pidCidCache *common.Cache) gosigarCidProvider {
	return gosigarCidProvider{
		log:                logp.NewLogger(providerName),
		hostPath:           hostPath,
		cgroupPrefixes:     cgroupPrefixes,
		cgroupRegex:        cgroupRegex,
		cidRegex:           regexp.MustCompile(`[\w]{64}`),
		processCgroupPaths: processCgroupPaths,
		pidCidCache:        pidCidCache,
	}
}

// getProcessCgroups returns a mapping of cgroup subsystem name to path. It
// returns an error if it failed to retrieve the cgroup info.
func (p gosigarCidProvider) getProcessCgroups(pid int) (cgroup.PathList, error) {

	var cgroup cgroup.PathList

	cgroup, err := p.processCgroupPaths(p.hostPath, pid)
	switch err.(type) {
	case nil, *os.PathError:
		// do no thing when err is nil or when os.PathError happens because the process don't exist,
		// or not running in linux system
	default:
		// should never happen
		return cgroup, errors.Wrapf(err, "failed to read cgroups for pid=%v", pid)
	}

	return cgroup, nil
}

// getCid checks all of the processes' paths to see if any
// of them are associated with Kubernetes. Kubernetes uses /kubepods/<quality>/<podId>/<cid> when
// naming cgroups and we use this to determine the container ID. If no container
// ID is found then an empty string is returned.
// Example:
// /kubepods/besteffort/pod9b9e44c2-00fd-11ea-95e9-080027421ddf/2bb9fd4de339e5d4f094e78bb87636004acfe53f5668104addc761fe4a93588e
// V2 Example:
// /kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod1f306eaea646903787fd7cc4eb6be515.slice/crio-eac98011dea91157038ca797f2141754106e074b7242721c7170a642a2204a54.scope
func (p gosigarCidProvider) getCid(cgroups cgroup.PathList) string {
	// if regex defined use it to find cid
	if len(p.cgroupRegex) != 0 {
		re := regexp.MustCompile(p.cgroupRegex)
		for _, path := range cgroups.Flatten() {
			rs := re.FindStringSubmatch(path.ControllerPath)
			if rs != nil {
				return rs[1]
			}
		}
	} else {
		// else, fall back to a hardcoded regex
		// In an attempt to not break the user-facing config interface for this processor,
		// fall back to the config'ed prefixes if we have cgv1, otherwise use regex
		// This should work with k8s on cgroupsV2, as we're still trying to extract the same container ID
		for _, path := range cgroups.Flatten() {
			if path.IsV2 {
				rs := p.cidRegex.FindStringSubmatch(path.ControllerPath)
				if len(rs) > 0 {
					return rs[0]
				}
			} else {
				for _, prefix := range p.cgroupPrefixes {
					if strings.HasPrefix(path.ControllerPath, prefix) {
						return filepath.Base(path.ControllerPath)
					}
				}
			}
		}
	}
	return ""
}
