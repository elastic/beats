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

package input_logfile

import (
	inputv2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Harvester collects the lines from a configured source. It is operated by the
// harvesterRunner, which opens a reading session per source and reads it in
// slices.
type Harvester interface {
	// Name returns the type of the Harvester.
	Name() string
	// Test checks if the Harvester can be started with the given configuration.
	Test(Source, inputv2.TestContext) error
	// OpenSession opens or resumes a reading session for the source, keeping the
	// source's file handle open across read slices. metrics is the shared input
	// metrics, updated as events are read.
	OpenSession(ctx inputv2.Context, src Source, cursor Cursor, metrics *Metrics) (HarvesterSession, error)
}

// SliceVerdict is the outcome of a single HarvesterSession.ReadSlice call,
// telling the runner what to do with the source next.
type SliceVerdict int

const (
	// SliceYield means no data is currently available (the read would block);
	// the reader parks the source until the waker sees new data.
	SliceYield SliceVerdict = iota
	// SliceDone means a terminal condition was reached (EOF on a closeable
	// file, truncation, error or cancellation); the source is torn down.
	SliceDone
)

// PollResult is the outcome of HarvesterSession.Poll, used by the runner's
// waker to decide what to do with a parked source.
type PollResult int

const (
	// PollPark means nothing changed; keep the source parked.
	PollPark PollResult = iota
	// PollResume means the source has new data; requeue it for reading.
	PollResume
	// PollClose means a close condition was met (inactive/removed/renamed);
	// tear the harvester down.
	PollClose
)

// HarvesterSession is an open reading session over a single source whose file
// handle stays open across many read slices, so a source can be resumed without
// being re-opened.
//
// Implementations are NOT safe for concurrent use: the runner guarantees a
// single goroutine operates a session at a time (one reader per source).
type HarvesterSession interface {
	// ReadSlice reads from the source and publishes events until there is no
	// data currently available (SliceYield) or a terminal condition is reached
	// (SliceDone).
	ReadSlice(ctx inputv2.Context, p Publisher) (SliceVerdict, error)
	// Poll is called by the runner's waker for a parked session to decide
	// whether to resume reading, keep parking, or close (inactive/removed/
	// renamed). It must not read or publish.
	Poll() PollResult
	// Offset returns the current read offset; the runner uses it to detect
	// whether a slice made progress.
	Offset() int64
	// IsGZIP reports whether the session reads a GZIP-compressed source, so the
	// runner can maintain the GZIP-specific lifecycle metrics.
	IsGZIP() bool
	// Close releases the file handle and resources held by the session.
	Close() error
}

// HarvesterGroup is responsible for running the Harvesters started by the
// Prospector.
type HarvesterGroup interface {
	// Start starts a Harvester for a Source.
	Start(inputv2.Context, Source)
	// Restart starts a Harvester if it might be already running.
	Restart(inputv2.Context, Source)
	// Continue starts a new Harvester with the state information of the previous.
	Continue(ctx inputv2.Context, previous, next Source)
	// Stop cancels the reader of a given Source.
	Stop(Source)
	// StopHarvesters cancels all running Harvesters.
	StopHarvesters() error
	// SetObserver sets the observer to get notified when a harvester closes.
	SetObserver(c chan HarvesterStatus)
}

// HarvesterStatus is used to notify an observer that the harvester for the ID
// has closed and the amount of data ingested from the file.
type HarvesterStatus struct {
	// ID is the ID of the harvester
	ID string
	// Size is the amount of data ingested, in other words the size of the file
	// when the harvester closed.
	Size int64
}

// lock locks a key for exclusive access and returns a resource that can be used
// to modify the cursor state and unlock the key.
func lock(ctx inputv2.Context, store *store, key string) (*resource, error) {
	resource := store.Get(key)
	err := lockResource(ctx.Logger, resource, ctx.Cancelation)
	if err != nil {
		resource.Release()
		return nil, err
	}

	resource.stateMutex.Lock()
	resource.lockedVersion = resource.version
	resource.stateMutex.Unlock()

	return resource, nil
}

func lockResource(log *logp.Logger, resource *resource, canceler inputv2.Canceler) error {
	if !resource.lock.TryLock() {
		log.Infof("Resource '%v' currently in use, waiting...", resource.key)
		err := resource.lock.LockContext(canceler)
		log.Infof("Resource '%v' finally released. Lock acquired", resource.key)
		if err != nil {
			log.Infof("Input for resource '%v' has been stopped while waiting", resource.key)
			return err
		}
	}
	return nil
}

func releaseResource(resource *resource) {
	resource.lock.Unlock()
	resource.Release()
}
