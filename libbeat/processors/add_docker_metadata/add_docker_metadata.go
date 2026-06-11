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

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
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

	pidFields       []string      // Field names that contain PIDs.
	cgroups         *common.Cache // Cache of PID (int) to container ids (string).
	dedot           bool          // If set to true, replace dots in labels with `_`.
	dockerAvailable atomic.Bool   // If Docker exists in env, then it is set to true
	closeRetry      chan struct{} // Channel to signal the connection retry goroutine to stop
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
	if container == nil {
		d.log.Debugf("Container not found: cid=%s", cid)
		return event, nil
	}

	event.Fields.DeepUpdate(d.buildContainerMeta(container))
	return event, nil
}

// RunPdata enriches the given pcommon.Map directly with Docker container metadata,
// avoiding the round-trip conversion to/from mapstr.M used by the standard Run path.
func (d *addDockerMetadata) RunPdata(body pcommon.Map) error {
	if !d.dockerAvailable.Load() {
		return nil
	}

	cid, err := d.resolveCIDFromPdata(body)
	if err != nil {
		return err
	}
	if cid == "" {
		return nil
	}

	container := d.watcher.Container(cid)
	if container == nil {
		d.log.Debugf("Container not found: cid=%s", cid)
		return nil
	}

	return otelmap.MergeMapstrIntoPdata(d.buildContainerMeta(container), body)
}

// resolveCIDFromPdata extracts the container ID from a pcommon.Map using the
// same three-path strategy as Run: sourceProcessor (log.file.path), PID-based
// cgroup lookup, and user-defined match fields. It also writes the resolved CID
// back into body under dockerContainerIDKey when found via the first two paths,
// mirroring what Run does on the beat.Event.
func (d *addDockerMetadata) resolveCIDFromPdata(body pcommon.Map) (string, error) {
	// Extract CID from log.file.path via sourceProcessor.
	if d.sourceProcessor != nil {
		if lfpVal, ok := otelmap.GetAtPath("log.file.path", body); ok && lfpVal.Type() == pcommon.ValueTypeStr {
			miniEvent := &beat.Event{Fields: mapstr.M{"log": mapstr.M{"file": mapstr.M{"path": lfpVal.Str()}}}}
			result, err := d.sourceProcessor.Run(miniEvent)
			if err != nil {
				d.log.Debugf("Error while extracting container ID from source path: %v", err)
			} else if result != nil {
				if v, err := result.GetValue(dockerContainerIDKey); err == nil {
					if cid, _ := v.(string); cid != "" {
						if err := otelmap.PutAtPath(dockerContainerIDKey, cid, body); err != nil {
							return "", err
						}
						return cid, nil
					}
				}
			}
		}
	}

	// Lookup CID via process cgroup membership.
	if len(d.pidFields) > 0 {
		miniEvent := &beat.Event{Fields: make(mapstr.M, len(d.pidFields))}
		for _, field := range d.pidFields {
			if v, ok := otelmap.GetAtPath(field, body); ok {
				_, _ = miniEvent.Fields.Put(field, v.AsRaw())
			}
		}
		id, err := d.lookupContainerIDByPID(miniEvent)
		if err != nil {
			return "", fmt.Errorf("error reading container ID: %w", err)
		}
		if id != "" {
			if err := otelmap.PutAtPath(dockerContainerIDKey, id, body); err != nil {
				return "", err
			}
			return id, nil
		}
	}

	// Lookup CID from user-defined match fields.
	for _, field := range d.fields {
		if v, ok := otelmap.GetAtPath(field, body); ok && v.Type() == pcommon.ValueTypeStr {
			return v.Str(), nil
		}
	}

	return "", nil
}

func (d *addDockerMetadata) buildContainerMeta(container *docker.Container) mapstr.M {
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
	return meta
}

func (d *addDockerMetadata) Close() error {
	d.closeOnce.Do(func() {
		if d.cgroups != nil {
			d.cgroups.StopJanitor()
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

		if d.cgroups != nil {
			if cid := d.cgroups.Get(pid); cid != nil {
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

		// Initialize at time of first use.
		lazyCgroupCacheInit(d)

		cid, err := getContainerIDFromCgroups(cgroups)
		d.cgroups.Put(pid, cid)

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
