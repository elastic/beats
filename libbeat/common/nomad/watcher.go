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

package nomad

import (
	"time"

	nomad "github.com/hashicorp/nomad/api"

	"github.com/elastic/beats/libbeat/logp"
)

// AllocationHandler can be implemented to set how to act when
// new allocations are started on the node
type AllocationHandler func(alloc *nomad.Allocation)

type watcher struct {
	client  NomadClient
	nodeID  string
	logger  *logp.Logger
	stopCh  chan struct{}
	handler AllocationHandler
}

// Watcher watches nomad allocations
type Watcher interface {
	// Start nomad API for new allocations
	Start() error

	// Stop watching nomad API for new allocations
	Stop()
}

// NewWatcherWithClient creates a new Watcher from a given Nomad client
func NewWatcherWithClient(client NomadClient, nodeID string, h AllocationHandler) (Watcher, error) {
	w := &watcher{
		client:  client,
		nodeID:  nodeID,
		logger:  logp.NewLogger("nomad"),
		stopCh:  make(chan struct{}),
		handler: h,
	}

	return w, nil
}

// Start watching nomad API for new allocations
func (w *watcher) Start() error {
	opts := &nomad.QueryOptions{WaitTime: 5 * time.Minute}

	for {
		allocs, meta, err := w.client.Allocations(w.nodeID, opts)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		select {
		case <-w.stopCh:
			return nil
		default:
		}

		// If no change in index we just cycle again
		if opts.WaitIndex == meta.LastIndex {
			continue
		}

		for _, alloc := range allocs {
			// If the allocation hasn't changed do nothing
			if opts.WaitIndex >= alloc.ModifyIndex {
				continue
			}
			w.handler(alloc)
		}
		opts.WaitIndex = meta.LastIndex
	}
}

// Stop watching the nomad API for new allocations
func (w *watcher) Stop() {
	close(w.stopCh)
}
