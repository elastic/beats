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

package add_docker_metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/gosigar/cgroup"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/docker"
	"github.com/elastic/beats/libbeat/common/safemapstr"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/actions"
)

const (
	processorName         = "add_docker_metadata"
	dockerContainerIDKey  = "docker.container.id"
	cgroupCacheExpiration = 5 * time.Minute
)

// processGroupPaths returns the cgroups associated with a process. This enables
// unit testing by allowing us to stub the OS interface.
var processCgroupPaths = cgroup.ProcessCgroupPaths

func init() {
	processors.RegisterPlugin(processorName, newDockerMetadataProcessor)
}

type addDockerMetadata struct {
	log             *logp.Logger
	watcher         docker.Watcher
	fields          []string
	sourceProcessor processors.Processor

	pidFields []string      // Field names that contain PIDs.
	cgroups   *common.Cache // Cache of PID (int) to cgropus (map[string]string).
	hostFS    string        // Directory where /proc is found
	dedot     bool          // If set to true, replace dots in labels with `_`.
}

func newDockerMetadataProcessor(cfg *common.Config) (processors.Processor, error) {
	return buildDockerMetadataProcessor(cfg, docker.NewWatcher)
}

func buildDockerMetadataProcessor(cfg *common.Config, watcherConstructor docker.WatcherConstructor) (processors.Processor, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrapf(err, "fail to unpack the %v configuration", processorName)
	}

	watcher, err := watcherConstructor(config.Host, config.TLS, config.MatchShortID)
	if err != nil {
		return nil, err
	}

	if err = watcher.Start(); err != nil {
		return nil, err
	}

	// Use extract_field processor to get container ID from source file path.
	var sourceProcessor processors.Processor
	if config.MatchSource {
		var procConf, _ = common.NewConfigFrom(map[string]interface{}{
			"field":     "source",
			"separator": "/",
			"index":     config.SourceIndex,
			"target":    "docker.container.id",
		})
		sourceProcessor, err = actions.NewExtractField(procConf)
		if err != nil {
			return nil, err
		}
	}

	return &addDockerMetadata{
		log:             logp.NewLogger(processorName),
		watcher:         watcher,
		fields:          config.Fields,
		sourceProcessor: sourceProcessor,
		pidFields:       config.MatchPIDs,
		hostFS:          config.HostFS,
		dedot:           config.DeDot,
	}, nil
}

func lazyCgroupCacheInit(d *addDockerMetadata) {
	if d.cgroups == nil {
		d.log.Debug("Initializing cgroup cache")
		evictionListener := func(k common.Key, v common.Value) {
			d.log.Debugf("Evicted cached cgroups for PID=%v", k)
		}
		d.cgroups = common.NewCacheWithRemovalListener(cgroupCacheExpiration, 100, evictionListener)
		d.cgroups.StartJanitor(5 * time.Second)
	}
}

func (d *addDockerMetadata) Run(event *beat.Event) (*beat.Event, error) {
	var cid string
	var err error

	// Extract CID from the filepath contained in the "source" field.
	if d.sourceProcessor != nil {
		if event.Fields["source"] != nil {
			event, err = d.sourceProcessor.Run(event)
			if err != nil {
				d.log.Debugf("Error while extracting container ID from source path: %v", err)
				return event, nil
			}

			if v, err := event.GetValue(dockerContainerIDKey); err == nil {
				cid, _ = v.(string)
			}
		}
	}

	// Lookup CID using process cgroup membership data.
	if cid == "" && len(d.pidFields) > 0 {
		if id := d.lookupContainerIDByPID(event); id != "" {
			cid = id
			event.PutValue(dockerContainerIDKey, cid)
		}
	}

	// Lookup CID from the user defined field names.
	if cid == "" && len(d.fields) > 0 {
		for _, field := range d.fields {
			value, err := event.GetValue(field)
			if err != nil {
				continue
			}

			if strValue, ok := value.(string); ok {
				cid = strValue
				break
			}
		}
	}

	if cid == "" {
		return event, nil
	}

	container := d.watcher.Container(cid)
	if container != nil {
		meta := common.MapStr{}
		metaIface, ok := event.Fields["docker"]
		if ok {
			meta = metaIface.(common.MapStr)
		}

		if len(container.Labels) > 0 {
			labels := common.MapStr{}
			for k, v := range container.Labels {
				if d.dedot {
					label := common.DeDot(k)
					labels.Put(label, v)
				} else {
					safemapstr.Put(labels, k, v)
				}
			}
			meta.Put("container.labels", labels)
		}

		meta.Put("container.id", container.ID)
		meta.Put("container.image", container.Image)
		meta.Put("container.name", container.Name)
		event.Fields["docker"] = meta.Clone()
	} else {
		d.log.Debugf("Container not found: cid=%s", cid)
	}

	return event, nil
}

func (d *addDockerMetadata) String() string {
	return fmt.Sprintf("%v=[match_fields=[%v] match_pids=[%v]]",
		processorName, strings.Join(d.fields, ", "), strings.Join(d.pidFields, ", "))
}

// lookupContainerIDByPID finds the container ID based on PID fields contained
// in the event.
func (d *addDockerMetadata) lookupContainerIDByPID(event *beat.Event) string {
	var cgroups map[string]string
	for _, field := range d.pidFields {
		v, err := event.GetValue(field)
		if err != nil {
			continue
		}

		pid, ok := common.TryToInt(v)
		if !ok {
			d.log.Debugf("field %v is not a PID (type=%T, value=%v)", field, v, v)
			continue
		}

		cgroups, err = d.getProcessCgroups(pid)
		if err != nil && os.IsNotExist(errors.Cause(err)) {
			continue
		}
		if err != nil {
			d.log.Debugf("failed to get cgroups for pid=%v: %v", pid, err)
		}

		break
	}

	return getContainerIDFromCgroups(cgroups)
}

// getProcessCgroups returns a mapping of cgroup subsystem name to path. It
// returns an error if it failed to retrieve the cgroup info.
func (d *addDockerMetadata) getProcessCgroups(pid int) (map[string]string, error) {
	// Initialize at time of first use.
	lazyCgroupCacheInit(d)

	cgroups, ok := d.cgroups.Get(pid).(map[string]string)
	if ok {
		d.log.Debugf("Using cached cgroups for pid=%v", pid)
		return cgroups, nil
	}

	cgroups, err := processCgroupPaths(d.hostFS, pid)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read cgroups for pid=%v", pid)
	}

	d.cgroups.Put(pid, cgroups)
	return cgroups, nil
}

// getContainerIDFromCgroups checks all of the processes' paths to see if any
// of them are associated with Docker. Docker uses /docker/<CID> when
// naming cgroups and we use this to determine the container ID. If no container
// ID is found then an empty string is returned.
func getContainerIDFromCgroups(cgroups map[string]string) string {
	for _, path := range cgroups {
		if strings.HasPrefix(path, "/docker") {
			return filepath.Base(path)
		}
	}

	return ""
}
