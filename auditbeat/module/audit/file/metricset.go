package file

import (
	"bytes"
	"os"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"

	"github.com/elastic/beats/auditbeat/datastore"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	metricsetName = "audit.file"
	logPrefix     = "[" + metricsetName + "]"
	bucketName    = metricsetName + ".v1"
)

var (
	debugf = logp.MakeDebug(metricsetName)
)

func init() {
	if err := mb.Registry.AddMetricSet("audit", "file", New, parse.EmptyHostParser); err != nil {
		panic(err)
	}
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

// MetricSet for monitoring file integrity.
type MetricSet struct {
	mb.BaseMetricSet
	config  Config
	reader  EventProducer
	scanner EventProducer

	// Runtime params that are initialized on Run().
	bucket       datastore.BoltBucket
	scanStart    time.Time
	scanChan     <-chan Event
	fsnotifyChan <-chan Event
}

// New returns a new file.MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v metricset is an experimental feature", metricsetName)

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	r, err := NewEventReader(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize audit file event reader")
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		config:        config,
		reader:        r,
	}

	if config.ScanAtStart {
		ms.scanner, err = NewFileSystemScanner(config)
		if err != nil {
			return nil, errors.Wrap(err, "failed to initialize audit file scanner")
		}
	}

	debugf("Initialized the audit file event reader. Running as euid=%v", os.Geteuid())

	return ms, nil
}

// Run runs the MetricSet. The method will not return control to the caller
// until it is finished (to stop it close the reporter.Done() channel).
func (ms *MetricSet) Run(reporter mb.PushReporter) {
	if !ms.init(reporter) {
		return
	}

	for ms.fsnotifyChan != nil || ms.scanChan != nil {
		select {
		case event, ok := <-ms.fsnotifyChan:
			if !ok {
				ms.fsnotifyChan = nil
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

func (ms *MetricSet) init(reporter mb.PushReporter) bool {
	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		err = errors.Wrap(err, "failed to open persistent datastore")
		reporter.Error(err)
		logp.Err("%v %v", logPrefix, err)
		return false
	}
	ms.bucket = bucket.(datastore.BoltBucket)

	ms.fsnotifyChan, err = ms.reader.Start(reporter.Done())
	if err != nil {
		err = errors.Wrap(err, "failed to start fsnotify event producer")
		reporter.Error(err)
		logp.Err("%v %v", logPrefix, err)
		return false
	}

	ms.scanStart = time.Now().UTC()
	if ms.scanner != nil {
		ms.scanChan, err = ms.scanner.Start(reporter.Done())
		if err != nil {
			err = errors.Wrap(err, "failed to start file scanner")
			reporter.Error(err)
			logp.Err("%v %v", logPrefix, err)
			return false
		}
	}

	return true
}

func (ms *MetricSet) reportEvent(reporter mb.PushReporter, event *Event) bool {
	if len(event.errors) > 0 && logp.IsDebug(metricsetName) {
		debugf("Errors on %v event for %v: %v",
			event.Action, event.Path, event.errors)
	}

	changed, lastEvent := ms.hasFileChangedSinceLastEvent(event)
	if changed {
		// Publish event if it changed.
		if ok := reporter.Event(buildMapStr(event, lastEvent != nil)); !ok {
			return false
		}
	}

	// Persist event locally.
	if event.Info == nil {
		if err := ms.bucket.Delete(event.Path); err != nil {
			logp.Err("%v %v", logPrefix, err)
		}
	} else {
		if err := store(ms.bucket, event); err != nil {
			logp.Err("%v %v", logPrefix, err)
		}
	}
	return true
}

func (ms *MetricSet) hasFileChangedSinceLastEvent(event *Event) (changed bool, lastEvent *Event) {
	// Load event from DB.
	lastEvent, err := load(ms.bucket, event.Path)
	if err != nil {
		logp.Warn("%v %v", logPrefix, err)
		return true, lastEvent
	}

	action, changed := diffEvents(lastEvent, event)
	if event.Action == 0 {
		event.Action = action
	}

	if changed && logp.IsDebug(metricsetName) {
		debugf("file at %v has changed since last seen: old=%v, new=%v",
			event.Path, lastEvent, event)
	}
	return changed, lastEvent
}

func (ms *MetricSet) purgeDeleted(reporter mb.PushReporter) {
	for _, prefix := range ms.config.Paths {
		deleted, err := purgeOlder(ms.bucket, ms.scanStart, prefix)
		if err != nil {
			logp.Err("%v %v", logPrefix, err)
			continue
		}

		for _, e := range deleted {
			// Don't persist!
			if !reporter.Event(buildMapStr(e, true)) {
				return
			}
		}
	}
}

// Datastore utility functions.

// purgeOlder does a prefix scan of the keys in the datastore and purges items
// older than the specified time.
func purgeOlder(b datastore.BoltBucket, t time.Time, prefix string) ([]*Event, error) {
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

	err := b.Update(func(b *bolt.Bucket) error {
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

	debugf("Purged %v of %v entries in %v for %v", len(deleted),
		totalKeys, time.Since(startTime), prefix)
	return deleted, err
}

// store stores and Event in the given Bucket.
func store(b datastore.Bucket, e *Event) error {
	builder, release := fbGetBuilder()
	defer release()
	data := fbEncodeEvent(builder, e)

	if err := b.Store(e.Path, data); err != nil {
		return errors.Wrapf(err, "failed to locally store event for %v", e.Path)
	}
	return nil
}

// load loads an Event from the datastore. It return a nil Event if the key was
// not found. It returns an error if there was a failure reading from the
// datastore or decoding the data.
func load(b datastore.Bucket, path string) (*Event, error) {
	var e *Event
	err := b.Load(path, func(blob []byte) error {
		e = fbDecodeEvent(path, blob)
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load locally persisted event for %v", path)
	}
	return e, nil
}
