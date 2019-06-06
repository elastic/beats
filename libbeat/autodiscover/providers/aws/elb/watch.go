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

package elb

import (
	"time"

	"github.com/elastic/beats/libbeat/logp"
)

type watcher struct {
	// gen tracks changes we increment the 'generation' of each entry in the map.
	gen         uint64
	fetcher     fetcher
	onStart     func(uuid string, lblMap *lbListener)
	onStop      func(uuid string)
	done        chan struct{}
	ticker      *time.Ticker
	lbListeners map[string]uint64
}

func newWatcher(
	fetcher fetcher,
	interval time.Duration,
	onStart func(uuid string, lblMap *lbListener),
	onStop func(uuid string)) *watcher {
	return &watcher{
		fetcher:     fetcher,
		onStart:     onStart,
		onStop:      onStop,
		done:        make(chan struct{}),
		ticker:      time.NewTicker(interval),
		lbListeners: map[string]uint64{},
	}
}

func (w *watcher) start() {
	go w.forever()
}

func (w *watcher) stop() {
	close(w.done)
}

func (w *watcher) forever() {
	for {
		select {
		case <-w.done:
			w.ticker.Stop()
		case <-w.ticker.C:
			err := w.once()
			if err != nil {
				logp.Err("error while fetching AWS ELBs: %s", err)
				return
			}
		}
	}
}

// once executes the watch loop a single time.
// This is mostly useful for testing.
func (w *watcher) once() error {
	fetchedLbls, err := w.fetcher.fetch()
	if err != nil {
		return err
	}

	oldGen := w.gen
	w.gen++

	// Increment the generation of all ELBs returned by the API request
	for _, lbl := range fetchedLbls {
		arn := lbl.arn()
		if _, exists := w.lbListeners[arn]; !exists {
			if w.onStart != nil {
				w.onStart(arn, lbl)
			}
		}
		w.lbListeners[arn] = w.gen
	}

	// ELBs not seen in the API request get deleted
	for uuid, entryGen := range w.lbListeners {
		if entryGen == oldGen {
			if w.onStop != nil {
				w.onStop(uuid)
			}
		}
	}

	return nil
}
