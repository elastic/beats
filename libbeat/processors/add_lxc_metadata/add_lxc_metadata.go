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

package add_lxc_metadata

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/gosigar/cgroup"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

const (
	processorName         = "add_lxc_metadata"
	lxcContainerIDKey     = "container.id"
	cgroupCacheExpiration = 5 * time.Minute
)

var lxcCgroupRegexp = regexp.MustCompile("^/lxc/([^/]+)")

// processGroupPaths returns the cgroups associated with a process.
// This enables unit testing by allowing us to stub the OS interface.
var processCgroupPaths = cgroup.ProcessCgroupPaths

func init() {
	processors.RegisterPlugin(processorName, New)
}

type addLxcMetadata struct {
	log       *logp.Logger
	cgroups   *common.Cache // Cache of PID (int) to cgropus (map[string]string).
	pidFields []string      // Field names that contain PIDs.
	hostFS    string        // Directory where /proc can be found.
}

// New constructs a new add_lxc_metadata processor.
func New(cfg *common.Config) (processors.Processor, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrapf(err, "Fail to unpack the %v configuration", processorName)
	}
	return &addLxcMetadata{
		log:       logp.NewLogger(processorName),
		pidFields: config.MatchPIDs,
		hostFS:    config.HostFS,
	}, nil
}

func lazyCgroupCacheInit(d *addLxcMetadata) {
	if d.cgroups == nil {
		d.log.Debug("Initializing cgroup cache")
		evictionListener := func(k common.Key, v common.Value) {
			d.log.Debugf("Evicted cached cgroups for PID=%v", k)
		}
		d.cgroups = common.NewCacheWithRemovalListener(cgroupCacheExpiration, 100, evictionListener)
		d.cgroups.StartJanitor(5 * time.Second)
	}
}

func (d *addLxcMetadata) Run(event *beat.Event) (*beat.Event, error) {
	// Lookup CID using process cgroup membership data.
	if len(d.pidFields) > 0 {
		if id := d.lookupContainerIDByPID(event); id != "" {
			event.PutValue(lxcContainerIDKey, id)

			meta := common.MapStr{}
			meta.Put("container.id", id)
			event.Fields.DeepUpdate(meta)
		}
	}
	return event, nil
}

func (d *addLxcMetadata) String() string {
	return fmt.Sprintf("%v=[match_pids=[%v]]",
		processorName, strings.Join(d.pidFields, ", "))
}

// lookupContainerIDByPID finds the container ID based on PID fields contained in the event.
func (d *addLxcMetadata) lookupContainerIDByPID(event *beat.Event) string {
	var cgroups map[string]string
	for _, field := range d.pidFields {
		v, err := event.GetValue(field)
		if err != nil {
			continue
		}

		pid, ok := common.TryToInt(v)
		if !ok {
			d.log.Debugf("Field %v is not a PID (type=%T, value=%v)", field, v, v)
			continue
		}

		cgroups, err = d.getProcessCgroups(pid)
		if err != nil && os.IsNotExist(errors.Cause(err)) {
			continue
		}
		if err != nil {
			d.log.Debugf("Failed to get cgroups for pid=%v: %v", pid, err)
		} else {
			d.log.Debugf("Cgroups for pid=%v found: %v", pid, cgroups)
		}

		break
	}
	return d.getContainerIDFromCgroups(cgroups)
}

// getProcessCgroups returns a mapping of cgroup subsystem name to path.
// It returns an error if it failed to retrieve the cgroup info.
func (d *addLxcMetadata) getProcessCgroups(pid int) (map[string]string, error) {
	// Initialize at time of first use.
	lazyCgroupCacheInit(d)

	cgroups, ok := d.cgroups.Get(pid).(map[string]string)
	if ok {
		d.log.Debugf("Using cached cgroups for pid=%v", pid)
		return cgroups, nil
	}

	cgroups, err := processCgroupPaths(d.hostFS, pid)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read cgroups for pid=%v: %v", pid, err)
	}

	d.cgroups.Put(pid, cgroups)
	return cgroups, nil
}

// getContainerIDFromCgroups checks all of the processes' paths to see if any
// of them are associated with LXC. LXC uses /lxc/<CID>/* when
// naming cgroups and we use this to determine the container ID.
// If no container ID is found then an empty string is returned.
func (d *addLxcMetadata) getContainerIDFromCgroups(cgroups map[string]string) string {
	for _, path := range cgroups {
		matches := lxcCgroupRegexp.FindStringSubmatch(path)
		if matches != nil {
			id := matches[1]
			d.log.Debugf("LXC container id detected in cgroup %v: %v", path, id)
			return id
		}
	}
	return ""
}
