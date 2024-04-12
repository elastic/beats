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

package file_integrity

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/elastic/beats/v7/auditbeat/datastore"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	moduleName    = "file_integrity"
	metricsetName = "file"
	bucketName    = "file.v1"

	// Use old namespace for data until we do some field renaming for GA.
	namespace = "."
)

var underTest bool //nolint:unused // Used in Darwin-only builds.

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
		mb.WithHostParser(parse.EmptyHostParser),
		mb.WithNamespace(namespace),
	)
}

// EventProducer produces events.
type EventProducer interface {
	// Start starts the event producer and writes events to the returned
	// channel. When the producer is finished it will close the returned
	// channel. If the returned event channel is not drained the producer will
	// block (possibly causing data loss). The producer can be stopped
	// prematurely by closing the provided done channel. An error is returned
	// if the producer fails to start.
	Start(done <-chan struct{}) (<-chan Event, error)
}

// eventProducerWithProcessor is an EventProducer that requires a Processor
type eventProducerWithProcessor interface {
	Processor() beat.Processor
}

// MetricSet for monitoring file integrity.
type MetricSet struct {
	mb.BaseMetricSet
	config  Config
	reader  EventProducer
	scanner EventProducer
	log     *logp.Logger

	// Runtime params that are initialized on Run().
	bucket    datastore.BoltBucket
	scanStart time.Time
	scanChan  <-chan Event
	eventChan <-chan Event

	// Used when a hash can't be calculated
	nullHashes map[HashType]Digest

	// Processors
	processors []beat.Processor
}

// New returns a new file.MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	logger := logp.NewLogger(moduleName)
	id := base.Module().Config().ID
	if id != "" {
		logger = logger.With("id", id)
	}

	r, err := NewEventReader(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize file event reader: %w", err)
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		config:        config,
		reader:        r,
		log:           logger,
	}

	// reader supports a processor
	if rWithProcessor, ok := r.(eventProducerWithProcessor); ok {
		if proc := rWithProcessor.Processor(); proc != nil {
			ms.processors = append(ms.processors, proc)
		}
	}

	ms.nullHashes = make(map[HashType]Digest, len(config.HashTypes))
	for _, hashType := range ms.config.HashTypes {
		// One byte is enough so that the hashes are persisted to the datastore.
		// The comparison function doesn't care if the lengths are not the expected
		// for the given algorithms.
		ms.nullHashes[hashType] = Digest{0x00}
	}
	ms.log.Debugf("Initialized the file event reader. Running as euid=%v", os.Geteuid())

	return ms, nil
}

func (ms *MetricSet) Processors() []beat.Processor {
	return ms.processors
}

// Run runs the MetricSet. The method will not return control to the caller
// until it is finished (to stop it close the reporter.Done() channel).
func (ms *MetricSet) Run(reporter mb.PushReporterV2) {
	if !ms.init(reporter) {
		return
	}

	for ms.eventChan != nil || ms.scanChan != nil {
		select {
		case event, ok := <-ms.eventChan:
			if !ok {
				ms.eventChan = nil
				continue
			}

			ms.reportEvent(reporter, &event)
		case event, ok := <-ms.scanChan:
			if !ok {
				ms.scanChan = nil
				// When the scan completes purge datastore keys that no longer
				// exist on disk based on being older than scanStart.
				ms.purgeDeleted(reporter)
				continue
			}

			ms.reportEvent(reporter, &event)
		case <-reporter.Done():
			return
		}
	}
}

// Close cleans up the MetricSet when it finishes.
func (ms *MetricSet) Close() error {
	if ms.bucket != nil {
		return ms.bucket.Close()
	}
	return nil
}

func (ms *MetricSet) init(reporter mb.PushReporterV2) bool {
	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		err = fmt.Errorf("failed to open persistent datastore: %w", err)
		reporter.Error(err)
		ms.log.Errorw("Failed to initialize", "error", err)
		return false
	}
	ms.bucket = bucket.(datastore.BoltBucket)

	ms.eventChan, err = ms.reader.Start(reporter.Done())
	if err != nil {
		err = fmt.Errorf("failed to start event producer: %w", err)
		reporter.Error(err)
		ms.log.Errorw("Failed to initialize", "error", err)
		return false
	}

	ms.scanStart = time.Now().UTC()
	if ms.config.ScanAtStart {
		ms.scanner, err = NewFileSystemScanner(ms.config, ms.findNewPaths())
		if err != nil {
			err = fmt.Errorf("failed to initialize file scanner: %w", err)
			reporter.Error(err)
			ms.log.Errorw("Failed to initialize", "error", err)
			return false
		}

		ms.scanChan, err = ms.scanner.Start(reporter.Done())
		if err != nil {
			err = fmt.Errorf("failed to start file scanner: %w", err)
			reporter.Error(err)
			ms.log.Errorw("Failed to initialize", "error", err)
			return false
		}
	}

	return true
}

// findNewPaths determines which - if any - paths have been newly added to the config.
func (ms *MetricSet) findNewPaths() map[string]struct{} {
	newPaths := make(map[string]struct{})

	for _, path := range ms.config.Paths {
		// Resolve symlinks and ensure we have an absolute path.
		evalPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			ms.log.Warnw("Failed to resolve symlink", "file_path", path, "error", err)
			continue
		}
		evalPath, err = filepath.Abs(evalPath)
		if err != nil {
			ms.log.Warnw("Failed to resolve to absolute path", "file_path", path, "error", err)
			continue
		}

		lastEvent, err := load(ms.bucket, evalPath)
		if err != nil {
			ms.log.Warnw("Failed during DB load", "error", err)
			continue
		}

		if lastEvent == nil {
			newPaths[evalPath] = struct{}{}
		}
	}

	return newPaths
}

func (ms *MetricSet) reportEvent(reporter mb.PushReporterV2, event *Event) bool {
	if len(event.errors) == 1 {
		ms.log.Debugw("Error in event", "file_path", event.Path,
			"action", event.Action, "error", event.errors[0])
	} else if len(event.errors) > 1 {
		ms.log.Debugw("Multiple errors in event", "file_path", event.Path,
			"action", event.Action, "errors", event.errors)
	}

	changed, lastEvent := ms.hasFileChangedSinceLastEvent(event)
	if changed {
		// Publish event if it changed.
		if ok := reporter.Event(buildMetricbeatEvent(event, lastEvent != nil)); !ok {
			return false
		}
	}

	// Persist event locally.
	if event.Info == nil {
		if err := ms.bucket.Delete(event.Path); err != nil {
			ms.log.Errorw("Failed during DB delete", "error", err)
		}
	} else {
		if event.hashFailed {
			// If hashing failed, persist the previous hashes, so it can detect
			// a future change to the file. Otherwise the next update event will
			// be reported as a config change.
			// Hashing usually fails while the file is being updated under Windows
			// if open in exclusive mode, and succeeds once the file is closed
			// and its mtime is updated.
			if lastEvent != nil {
				event.Hashes = lastEvent.Hashes
			} else {
				event.Hashes = ms.nullHashes
			}
		}
		if err := store(ms.bucket, event); err != nil {
			ms.log.Errorw("Failed during DB store", "error", err)
		}
	}
	return true
}

func (ms *MetricSet) hasFileChangedSinceLastEvent(event *Event) (changed bool, lastEvent *Event) {
	// Load event from DB.
	lastEvent, err := load(ms.bucket, event.Path)
	if err != nil {
		ms.log.Warnw("Failed during DB load", "error", err)
		return true, lastEvent
	}

	// Received a deleted event but the file now exists on disk (already re-created).
	if event.Action&Deleted != 0 && event.Info != nil {
		event.Action &= ^Action(Deleted)
		event.Action |= Updated
	}
	// We receive a creation event for a deletion that we didn't observe due to the above.
	if event.Action&Created != 0 && lastEvent != nil && lastEvent.Info != nil {
		event.Action &= ^Action(Created)
		event.Action |= Updated
	}
	action, changed := diffEvents(lastEvent, event)
	if uint8(event.Action)&^uint8(Updated) == 0 {
		if event.hashFailed && !changed {
			event.Action = Updated
		} else {
			event.Action = action
		}
		changed = event.Action != None
	}
	if changed {
		ms.log.Debugw("File changed since it was last seen",
			"file_path", event.Path, "took", event.rtt,
			logp.Namespace("event"), "action", event.Action,
			"old", lastEvent, "new", event)
	}
	return changed, lastEvent
}

func (ms *MetricSet) purgeDeleted(reporter mb.PushReporterV2) {
	for _, prefix := range ms.config.Paths {
		deleted, err := ms.purgeOlder(ms.scanStart, prefix)
		if err != nil {
			ms.log.Errorw("Failure while purging older records", "error", err)
			continue
		}

		for _, e := range deleted {
			// Don't persist!
			if !ms.config.IsExcludedPath(e.Path) {
				reporter.Event(buildMetricbeatEvent(e, true))
			}
		}
	}
}

// Datastore utility functions.

// purgeOlder does a prefix scan of the keys in the datastore and purges items
// older than the specified time.
func (ms *MetricSet) purgeOlder(t time.Time, prefix string) ([]*Event, error) {
	var (
		deleted       []*Event
		totalKeys     uint64
		p             = []byte(prefix)
		matchesPrefix = func(path []byte) bool {
			// XXX: This match may need to be smarter to accommodate multiple
			// metricset instances working on similar paths (e.g. /a and /a/b)
			// or when recursion is allowed.
			return bytes.HasPrefix(path, p)
		}
		startTime = time.Now()
	)

	err := ms.bucket.Update(func(b *bolt.Bucket) error {
		c := b.Cursor()

		for path, v := c.Seek(p); path != nil && matchesPrefix(path); path, v = c.Next() {
			totalKeys++

			if fbIsEventTimestampBefore(v, t) {
				if err := c.Delete(); err != nil {
					return err
				}

				deleted = append(deleted, &Event{
					Timestamp: t,
					Action:    Deleted,
					Path:      string(path),
				})
			}
		}
		return nil
	})

	took := time.Since(startTime)
	ms.log.With(
		"file_path", prefix,
		"took", took,
		"items_total", totalKeys,
		"items_deleted", len(deleted)).
		Debugf("Purged %v of %v entries in %v for %v", len(deleted), totalKeys,
			time.Since(startTime), prefix)
	return deleted, err
}

// store stores and Event in the given Bucket.
func store(b datastore.Bucket, e *Event) error {
	builder, release := fbGetBuilder()
	defer release()
	data := fbEncodeEvent(builder, e)

	if err := b.Store(e.Path, data); err != nil {
		return fmt.Errorf("failed to locally store event for %v: %w", e.Path, err)
	}
	return nil
}

// load loads an Event from the datastore. It returns a nil Event if the key was
// not found. It returns an error if there was a failure reading from the
// datastore or decoding the data.
func load(b datastore.Bucket, path string) (*Event, error) {
	var e *Event
	err := b.Load(path, func(blob []byte) error {
		e = fbDecodeEvent(path, blob)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load locally persisted event for %v: %w", path, err)
	}
	return e, nil
}
