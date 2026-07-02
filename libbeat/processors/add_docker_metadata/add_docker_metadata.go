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

//go:build linux || darwin || windows

package add_docker_metadata

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/actions"
	"github.com/elastic/elastic-agent-autodiscover/docker"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/safemapstr"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

const (
	processorName         = "add_docker_metadata"
	dockerContainerIDKey  = "container.id"
	cgroupCacheExpiration = 5 * time.Minute
)

// initCgroupPaths initializes a new cgroup reader. This enables
// unit testing by allowing us to stub the OS interface.
var initCgroupPaths processors.InitCgroupHandler = func(rootfsMountpoint resolve.Resolver, ignoreRootCgroups bool) (processors.CGReader, error) {
	return cgroup.NewReaderOptions(cgroup.ReaderOptions{
		RootfsMountpoint:  rootfsMountpoint,
		IgnoreRootCgroups: ignoreRootCgroups,
	})
}

func init() {
	processors.RegisterPlugin(processorName, New)
}

type addDockerMetadata struct {
	log             *logp.Logger
	watcher         docker.Watcher
	fields          []string
	sourceProcessor beat.Processor

	pidFields       []string                     // Field names that contain PIDs.
	cgroups         atomic.Pointer[common.Cache] // Cache of PID (int) to container ids (string).
	cgroupsOnce     sync.Once                    // Guards the lazy initialization of cgroups.
	dedot           bool                         // If set to true, replace dots in labels with `_`.
	dockerAvailable atomic.Bool                  // If Docker exists in env, then it is set to true
	closeRetry      chan struct{}                // Channel to signal the connection retry goroutine to stop
	waitRetry       sync.WaitGroup
	closeOnce       sync.Once
	closeErr        error
	cgreader        processors.CGReader
	retryPeriod     time.Duration // Period to wait when reconnecting to Docker
	retryTimeout    time.Duration // Maximum time to wait when connecting to Docker, 0 means wait forever.
	retryIsBlocking bool          // If true, startup waits for the retry loop before returning.
}

const selector = "add_docker_metadata"

// New constructs a new add_docker_metadata processor.
func New(cfg *conf.C, log *logp.Logger) (beat.Processor, error) {
	return buildDockerMetadataProcessor(log.Named(selector), cfg, docker.NewWatcher)
}

func buildDockerMetadataProcessor(log *logp.Logger, cfg *conf.C, watcherConstructor docker.WatcherConstructor) (beat.Processor, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("fail to unpack the %v configuration: %w", processorName, err)
	}

	// Use extract_field processor to get container ID from source file path.
	var sourceProcessor beat.Processor
	var err error
	if config.MatchSource {
		var procConf, _ = conf.NewConfigFrom(map[string]interface{}{
			"field":     "log.file.path",
			"separator": string(os.PathSeparator),
			"index":     config.SourceIndex,
			"target":    dockerContainerIDKey,
		})
		sourceProcessor, err = actions.NewExtractField(procConf, log)
		if err != nil {
			return nil, err
		}
	}

	reader, err := initCgroupPaths(resolve.NewTestResolver(config.HostFS), false)
	if errors.Is(err, cgroup.ErrCgroupsMissing) {
		reader = &processors.NilCGReader{}
	} else if err != nil {
		return nil, fmt.Errorf("error creating cgroup reader: %w", err)
	}

	dm := addDockerMetadata{
		log:             log,
		fields:          config.Fields,
		sourceProcessor: sourceProcessor,
		pidFields:       config.MatchPIDs,
		dedot:           config.DeDot,
		cgreader:        reader,
		closeRetry:      make(chan struct{}),
		retryPeriod:     config.WaitMetadataRetry,
		retryTimeout:    config.WaitMetadataTimeout,
		retryIsBlocking: config.WaitMetadata,
	}

	constructAndStartWatcher := func() (docker.Watcher, error) {
		watcher, err := watcherConstructor(log, config.Host, config.TLS, config.MatchShortID)
		if err != nil {
			log.Debugf("%v: docker environment not detected: %+v", processorName, err)
			return nil, err
		}

		log.Debugf("%v: docker environment detected", processorName)
		if err = watcher.Start(); err != nil {
			log.Debugf("unable to start the docker watcher: %v", err)
			return nil, err
		}

		log.Info("successfully connected to docker")
		return watcher, nil
	}

	connectToDocker := func() error {
		watcher, err := constructAndStartWatcher()
		if err != nil {
			return err
		}

		dm.watcher = watcher
		dm.dockerAvailable.Store(true)
		return nil
	}

	retryStart := time.Now()
	if err := connectToDocker(); err != nil {
		if dm.retryIsBlocking {
			connected, _, lastErr := dm.retryConnectToDocker(connectToDocker, retryStart)
			if !connected {
				if err := processors.Close(dm.sourceProcessor); err != nil {
					dm.log.Debugf("error closing source processor after docker connection timeout: %v", err)
				}
				if lastErr == nil {
					lastErr = err
				}
				return nil, fmt.Errorf("%s: could not connect to docker: %w", processorName, lastErr)
			}
		} else {
			// If docker is not available, try reconnecting asynchronously until the timeout expires.
			dm.startDockerConnectionRetry(connectToDocker, retryStart)
		}
	}

	return &dm, nil
}

func (d *addDockerMetadata) startDockerConnectionRetry(connectToDocker func() error, retryStart time.Time) {
	d.waitRetry.Go(func() {
		defer d.log.Debug("retry goroutine done")
		connected, stopped, _ := d.retryConnectToDocker(connectToDocker, retryStart)
		if !connected && !stopped {
			d.log.Warnf(
				"stopped retrying docker connection before metadata became available; wait_for_metadata_timeout=%s elapsed",
				d.retryTimeout,
			)
		}
	})
}

func (d *addDockerMetadata) retryConnectToDocker(connectToDocker func() error, retryStart time.Time) (connected bool, stopped bool, lastErr error) {
	blockingStr := "non-blocking"
	if d.retryIsBlocking {
		blockingStr = "blocking"
	}

	d.log.Warnf(
		"could not connect to docker, retrying (%s) connection attempts every %s "+
			"with a maximum wait of %s (0 means indefinitely)",
		blockingStr,
		d.retryPeriod,
		d.retryTimeout,
	)

	ticker := time.NewTicker(d.retryPeriod)
	defer ticker.Stop()

	var timeoutC <-chan time.Time
	var timeoutTimer *time.Timer
	if d.retryTimeout > 0 {
		remaining := time.Until(retryStart.Add(d.retryTimeout))
		if remaining <= 0 {
			return false, false, nil
		}
		timeoutTimer = time.NewTimer(remaining)
		timeoutC = timeoutTimer.C
		defer timeoutTimer.Stop()
	}

	for {
		select {
		case <-ticker.C:
			if err := connectToDocker(); err != nil {
				lastErr = err
			} else {
				return true, false, nil
			}
		case <-timeoutC:
			return false, false, lastErr
		case <-d.closeRetry:
			return false, true, lastErr
		}
	}
}

// cgroupCache returns the PID-to-container-ID cache, creating it and starting
// its janitor on first use. It is safe to call from concurrent Run goroutines.
func (d *addDockerMetadata) cgroupCache() *common.Cache {
	d.cgroupsOnce.Do(func() {
		d.log.Debug("Initializing cgroup cache")
		evictionListener := func(k common.Key, v common.Value) {
			d.log.Debugf("Evicted cached cgroups for PID=%v", k)
		}
		cache := common.NewCacheWithRemovalListener(cgroupCacheExpiration, 100, evictionListener)
		cache.StartJanitor(5 * time.Second)
		d.cgroups.Store(cache)
	})
	return d.cgroups.Load()
}

func (d *addDockerMetadata) Run(event *beat.Event) (*beat.Event, error) {
	if !d.dockerAvailable.Load() {
		return event, nil
	}
	var cid string
	var err error

	// Extract CID from the filepath contained in the "log.file.path" field.
	if d.sourceProcessor != nil {
		lfp, _ := event.Fields.GetValue("log.file.path")
		if lfp != nil {
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
		id, err := d.lookupContainerIDByPID(event)
		if err != nil {
			return nil, fmt.Errorf("error reading container ID: %w", err)
		}
		if id != "" {
			cid = id
			_, _ = event.PutValue(dockerContainerIDKey, cid)
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
		meta := mapstr.M{}

		if len(container.Labels) > 0 {
			labels := mapstr.M{}
			for k, v := range container.Labels {
				if d.dedot {
					label := common.DeDot(k)
					_, _ = labels.Put(label, v)
				} else {
					_ = safemapstr.Put(labels, k, v)
				}
			}
			_, _ = meta.Put("container.labels", labels)
		}

		_, _ = meta.Put("container.id", container.ID)
		_, _ = meta.Put("container.image.name", container.Image)
		_, _ = meta.Put("container.name", container.Name)
		event.Fields.DeepUpdate(meta)
	} else {
		d.log.Debugf("Container not found: cid=%s", cid)
	}

	return event, nil
}

func (d *addDockerMetadata) Close() error {
	d.closeOnce.Do(func() {
		if cgroups := d.cgroups.Load(); cgroups != nil {
			cgroups.StopJanitor()
		}

		// Stop the retry goroutine, this is safe to call even if the goroutine is not running.
		close(d.closeRetry)
		d.waitRetry.Wait()

		// If the watcher is running, stop it.
		if d.dockerAvailable.Load() && d.watcher != nil {
			d.watcher.Stop()
		}

		err := processors.Close(d.sourceProcessor)
		if err != nil {
			d.closeErr = fmt.Errorf("closing source processor of add_docker_metadata: %w", err)
		}
	})
	return d.closeErr
}

func (d *addDockerMetadata) String() string {
	return fmt.Sprintf("%v=[match_fields=[%v] match_pids=[%v]]",
		processorName, strings.Join(d.fields, ", "), strings.Join(d.pidFields, ", "))
}

// lookupContainerIDByPID finds the container ID based on PID fields contained
// in the event.
func (d *addDockerMetadata) lookupContainerIDByPID(event *beat.Event) (string, error) {
	pids := make([]int, 0, len(d.pidFields))

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

		if cgroups := d.cgroups.Load(); cgroups != nil {
			if cid := cgroups.Get(pid); cid != nil {
				d.log.Debugf("Using cached cgroups for pid=%v", pid)
				cidStr, ok := cid.(string)
				if !ok {
					d.log.Debugf("cached cgroup value for pid=%v is not a string (type=%T)", pid, cid)
					continue
				}
				return cidStr, nil
			}
		}

		pids = append(pids, pid)
	}

	for _, pid := range pids {
		cgroups, err := d.getProcessCgroups(pid)
		if err != nil && errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			d.log.Debugf("failed to get cgroups for pid=%v: %v", pid, err)
		}

		cid, err := getContainerIDFromCgroups(cgroups)
		// Cache the result, creating the cache on first use.
		d.cgroupCache().Put(pid, cid)

		return cid, err
	}

	return "", nil
}

// getProcessCgroups returns a mapping of cgroup subsystem name to path. It
// returns an error if it failed to retrieve the cgroup info.
func (d *addDockerMetadata) getProcessCgroups(pid int) (cgroup.PathList, error) {
	cgroups, err := d.cgreader.ProcessCgroupPaths(pid)
	if err != nil {
		return cgroups, fmt.Errorf("failed to read cgroups for pid=%v: %w", pid, err)
	}
	if len(cgroups.Flatten()) == 0 {
		return cgroup.PathList{}, fs.ErrNotExist
	}
	return cgroups, nil
}

var re = regexp.MustCompile(`[\w]{64}`)

// getContainerIDFromCgroups checks all of the processes' paths to see if any
// of them are associated with Docker. For cgroups V1, Docker uses /docker/<CID> when
// naming cgroups and we use this to determine the container ID. For V2,
// it's part of a more complex string.
func getContainerIDFromCgroups(cgroups cgroup.PathList) (string, error) {
	for _, path := range cgroups.Flatten() {
		rs := re.FindStringSubmatch(path.ControllerPath)
		if rs != nil {
			return rs[0], nil
		}
	}

	return "", nil
}
