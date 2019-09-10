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
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/logp"
	api "github.com/hashicorp/nomad/api"
)

// Max back off time for retries
const maxBackoff = 30 * time.Second

// Watcher watches Nomad task events
type Watcher interface {
	// Start watching Nomad API for new events
	Start() error

	// Stop watching Nomad API for new events
	Stop()

	// AddEventHandler add event handlers for handling specific events
	AddEventHandler(ResourceEventHandlerFuncs)
}

type watcher struct {
	client    *api.Client
	options   WatchOptions
	logger    *logp.Logger
	nodeID    string
	lastFetch time.Time
	handler   ResourceEventHandlerFuncs
}

// WatchOptions controls watch behaviors
type WatchOptions struct {
	// SyncTimeout is a timeout for tasks
	SyncTimeout time.Duration
	// Node is used for filtering events
	Node string
	// Namespace is used for filtering events on specified namespacesx,
	Namespace string
	// RefreshInterval is the time interval that the Nomad API will be queried
	RefreshInterval time.Duration
}

// NewWatcher initializes the watcher client to provide a events handler for
// resource from the cluster (filtered to the given node)
func NewWatcher(client *api.Client, options WatchOptions) (Watcher, error) {
	w := &watcher{
		client:  client,
		options: options,
		logger:  logp.NewLogger("nomad"),
	}

	return w, nil
}

func (w *watcher) Start() error {
	// Get the initial annotations and metadata
	err := w.sync()
	if err != nil {
		w.Stop()
		return err
	}

	// initiate the watcher
	go w.watch()

	return nil
}

func (w *watcher) Stop() {}

func (w *watcher) AddEventHandler(h ResourceEventHandlerFuncs) {
	w.handler = h
}

// Sync the allocations on the given node and update the local metadata
func (w *watcher) sync() error {
	w.logger.Info("Nomad: Syncing allocations and metadata")

	queryOpts := &api.QueryOptions{
		WaitTime:   w.options.SyncTimeout,
		AllowStale: true,
		WaitIndex:  1,
	}

	if w.nodeID == "" {
		// Fetch the nodeId from the node name, used to filter the allocations
		// If for some reason the NodeID changes filebeat will have to be restarted as well
		nodes, _, err := w.client.Nodes().List(queryOpts)
		if err != nil {
			w.logger.Errorf("Nomad: Fetching node list err %s", err.Error())
			return err
		}

		for _, node := range nodes {
			if node.Name == w.options.Node {
				w.nodeID = node.ID
				break
			}
		}
	}

	// Do we need to keep direct access to the metadata as well?
	allocations, meta, err := w.client.Nodes().Allocations(w.nodeID, queryOpts)
	if err != nil {
		w.logger.Errorf("Nomad: Fetching allocations err %s for %s,%s", err.Error(), w.options.Node, w.nodeID)
		return err
	}

	remoteWaitIndex := meta.LastIndex
	localWaitIndex := queryOpts.WaitIndex

	// Only work if the WaitIndex have changed
	if remoteWaitIndex == localWaitIndex {
		w.logger.Debug("Allocations index is unchanged (%d == %d)",
			fmt.Sprint(remoteWaitIndex), fmt.Sprint(localWaitIndex))
		return nil
	}

	for _, alloc := range allocations {
		// "Patch" the local hostname/node name into the allocations. filebeat
		// runs locally on each client filters the allocations based on the
		// hostname/client node name. Due this particular setup all allocations
		// fetched from the API are coming from the same client.
		// We patch the NodeName property if empty (Nomad < 0.9) to avoid
		// fetching it through the API
		if len(alloc.NodeName) == 0 {
			alloc.NodeName = w.options.Node
		}

		switch alloc.DesiredStatus {
		case AllocDesiredStatusRun:
			w.handler.OnAdd(*alloc)
		case AllocDesiredStatusStop:
			w.handler.OnDelete(*alloc)
		case AllocDesiredStatusEvict:
			w.handler.OnDelete(*alloc)
		}

		// allocation was updated after our last fetch
		if alloc.ModifyTime > w.lastFetch.Unix() {
			w.handler.OnUpdate(*alloc)
		}
	}

	w.logger.Debug("Allocations index has changed (%d != %d)",
		fmt.Sprint(remoteWaitIndex), fmt.Sprint(localWaitIndex))

	queryOpts.WaitIndex = meta.LastIndex
	w.lastFetch = time.Now()

	return nil
}

func (w *watcher) watch() {
	// Failures counter, do exponential backoff on retries
	var failures uint
	logp.Info("Nomad: %s", "Watching API for resource events")
	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ticker.C:
			err := w.sync()
			if err != nil {
				logp.Err("Nomad: Error while watching for allocation changes %v", err)
				backoff(failures)
				failures++
			}
		}
	}
}

func backoff(failures uint) {
	wait := 1 << failures * time.Second
	if wait > maxBackoff {
		wait = maxBackoff
	}

	time.Sleep(wait)
}
