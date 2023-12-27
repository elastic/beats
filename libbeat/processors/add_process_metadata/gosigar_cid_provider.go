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
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

const (
	providerName = "gosigar_cid_provider"
)

type gosigarCidProvider struct {
	log                *logp.Logger
	hostPath           resolve.Resolver
	cgroupPrefixes     []string
	cgroupRegex        *regexp.Regexp
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

	cid = p.getContainerID(cgroups)

	// add pid and cid to cache
	if p.pidCidCache != nil {
		p.pidCidCache.Put(pid, cid)
	}
	return cid, nil
}

func newCidProvider(hostPath resolve.Resolver, cgroupPrefixes []string, cgroupRegex *regexp.Regexp, processCgroupPaths func(resolve.Resolver, int) (cgroup.PathList, error), pidCidCache *common.Cache) gosigarCidProvider {
	return gosigarCidProvider{
		log:                logp.NewLogger(providerName),
		hostPath:           hostPath,
		cgroupPrefixes:     cgroupPrefixes,
		cgroupRegex:        cgroupRegex,
		processCgroupPaths: processCgroupPaths,
		pidCidCache:        pidCidCache,
	}
}

// getProcessCgroups returns a mapping of cgroup subsystem name to path. It
// returns an error if it failed to retrieve the cgroup info.
func (p gosigarCidProvider) getProcessCgroups(pid int) (cgroup.PathList, error) {
	pathList, err := p.processCgroupPaths(p.hostPath, pid)
	if err != nil {
		var pathError *fs.PathError
		if errors.As(err, &pathError) {
			// do no thing when err is nil or when os.PathError happens because the process don't exist,
			// or not running in linux system
			return cgroup.PathList{}, nil
		}
		// should never happen
		return cgroup.PathList{}, fmt.Errorf("failed to read cgroups for pid=%v: %w", pid, err)
	}

	return pathList, nil
}

// getContainerID checks all the processes' cgroup paths to see if any match the
// configured cgroup_regex or cgroup_prefixes. If there is a match, then the
// container ID is returned. Otherwise, an empty string is returned.
func (p gosigarCidProvider) getContainerID(cgroups cgroup.PathList) string {
	if p.cgroupRegex != nil {
		for _, path := range cgroups.Flatten() {
			rs := p.cgroupRegex.FindStringSubmatch(path.ControllerPath)
			if len(rs) > 1 {
				return rs[1]
			}
		}
		return ""
	}

	// Try cgroup_prefixes.
	for _, path := range cgroups.Flatten() {
		for _, prefix := range p.cgroupPrefixes {
			if strings.HasPrefix(path.ControllerPath, prefix) {
				return filepath.Base(path.ControllerPath)
			}
		}
	}
	return ""
}
