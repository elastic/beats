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

package filestream

import (
	"time"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

// testWatcher wraps a fileWatcher and collects the events its scans produce
// on a channel, so tests can read them the way the prospector's sink would.
type testWatcher struct {
	*fileWatcher
	events chan loginp.FSEvent
}

func wrapTestWatcher(w *fileWatcher) *testWatcher {
	return &testWatcher{fileWatcher: w, events: make(chan loginp.FSEvent, 128)}
}

func newTestFileWatcher(
	logger *logp.Logger,
	paths []string,
	config fileWatcherConfig,
	compression string,
	sendNotChanged bool,
	fi fileIdentifier,
	srci *loginp.SourceIdentifier,
) (*testWatcher, error) {
	w, err := newFileWatcher(logger, paths, config, compression, sendNotChanged, fi, srci)
	if err != nil {
		return nil, err
	}
	return wrapTestWatcher(w), nil
}

func (w *testWatcher) sink(ctx unison.Canceler) loginp.FSEventSink {
	return func(e loginp.FSEvent) {
		select {
		case w.events <- e:
		case <-ctx.Done():
		}
	}
}

// watch runs one scan, sending its events to w.events.
func (w *testWatcher) watch(ctx unison.Canceler, metrics *loginp.Metrics, ignoreOlder time.Duration, ignoreInactiveSince time.Time) {
	w.ScanOnce(ctx, w.sink(ctx), metrics, ignoreOlder, ignoreInactiveSince)
}

// Run scans every check interval until ctx is cancelled, like the prospector,
// then closes w.events so Event returns an OpDone event.
func (w *testWatcher) Run(ctx unison.Canceler, metrics *loginp.Metrics, ignoreOlder time.Duration, ignoreInactiveSince time.Time) {
	defer close(w.events)
	defer metrics.Cleanup()
	w.watch(ctx, metrics, ignoreOlder, ignoreInactiveSince)
	if w.cfg.Interval <= 0 {
		return
	}
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			w.watch(ctx, metrics, ignoreOlder, ignoreInactiveSince)
		case <-ctx.Done():
			return
		}
	}
}

// Event returns the next event a scan produced.
func (w *testWatcher) Event() loginp.FSEvent {
	return <-w.events
}
