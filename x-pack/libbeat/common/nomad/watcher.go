// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package nomad

import (
	"fmt"
	"time"

	api "github.com/hashicorp/nomad/api"

	"github.com/elastic/beats/libbeat/logp"
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
	waitIndex uint64
	done      chan struct{}
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
	// AllowStale allows any Nomad server (non-leader) to service
	// a read. This allows for lower latency and higher throughput
	AllowStale bool
	// InitialWaitIndex specify the initial WaitIndex to send in the first request
	// to the Nomad API
	InitialWaitIndex uint64
}

// NewWatcher initializes the watcher client to provide a events handler for
// resource from the cluster (filtered to the given node)
func NewWatcher(client *api.Client, options WatchOptions) (Watcher, error) {
	w := &watcher{
		client:    client,
		options:   options,
		logger:    logp.NewLogger("nomad"),
		waitIndex: options.InitialWaitIndex,
		lastFetch: time.Now(),
		done:      make(chan struct{}),
	}

	queryOpts := &api.QueryOptions{
		WaitTime:   w.options.SyncTimeout,
		AllowStale: w.options.AllowStale,
		WaitIndex:  uint64(1),
	}

	if w.nodeID == "" {
		// Fetch the nodeId from the node name, used to filter the allocations
		// If for some reason the NodeID changes filebeat will have to be restarted
		nodes, _, err := w.client.Nodes().List(queryOpts)

		if err != nil {
			w.logger.Fatalf("Nomad: Fetching node list err %s", err.Error())
			return nil, err
		}

		for _, node := range nodes {
			if node.Name == w.options.Node {
				w.nodeID = node.ID
				break
			}
		}
	}

	return w, nil
}

func (w *watcher) Start() error {
	// Get the initial annotations and metadata
	err := w.sync()
	if err != nil {
		return err
	}

	// initiate the watcher
	go w.watch()

	return nil
}

func (w *watcher) Stop() {
	close(w.done)
}

func (w *watcher) AddEventHandler(h ResourceEventHandlerFuncs) {
	w.handler = h
}

// Sync the allocations on the given node and update the local metadata
func (w *watcher) sync() error {
	w.logger.Info("Nomad: Syncing allocations and metadata")
	w.logger.Debugf("Starting with WaitIndex=%v", w.waitIndex)

	queryOpts := &api.QueryOptions{
		WaitTime:   w.options.SyncTimeout,
		AllowStale: w.options.AllowStale,
		WaitIndex:  w.waitIndex,
	}

	w.logger.Infof("Filtering allocations running in node: [%s, %s]", w.options.Node, w.nodeID)

	// Do we need to keep direct access to the metadata as well?
	allocations, meta, err := w.client.Nodes().Allocations(w.nodeID, queryOpts)
	if err != nil {
		w.logger.Errorf("Nomad: Fetching allocations err %s for %s,%s", err.Error(), w.options.Node, w.nodeID)
		return err
	}

	w.logger.Infof("Found %d allocations", len(allocations))

	remoteWaitIndex := meta.LastIndex
	localWaitIndex := queryOpts.WaitIndex

	// Only emit updated metadata if the WaitIndex have changed
	if remoteWaitIndex <= localWaitIndex {
		w.logger.Debugf("Allocations index is unchanged remoteWaitIndex=%v == localWaitIndex=%v",
			fmt.Sprint(remoteWaitIndex), fmt.Sprint(localWaitIndex))
		return nil
	}

	for _, alloc := range allocations {
		if alloc.ModifyIndex < w.waitIndex {
			continue
		}

		// "patch" the local hostname/node name into the allocations. filebeat
		// runs locally on each client filters the allocations based on the
		// hostname/client node name. Due this particular setup all allocations
		// fetched from the API are coming from the same client.
		// We patch the NodeName property if empty (Nomad < 0.9) to avoid
		// fetching it through the API
		if len(alloc.NodeName) == 0 {
			alloc.NodeName = w.options.Node
		}

		w.logger.Debugf("Received allocation: %s DesiredStatus:%s ClientStatus:%s", alloc.ID,
			alloc.DesiredStatus, alloc.ClientStatus)

		switch alloc.ClientStatus {
		case AllocClientStatusComplete, AllocClientStatusFailed, api.AllocClientStatusLost:
			w.handler.OnDelete(*alloc)
		case AllocClientStatusRunning:
			// Handle in-place allocation updates (like adding tags to a service definition) that
			// don't trigger a new allocation
			if (alloc.CreateIndex < alloc.AllocModifyIndex) && (w.waitIndex > 1) {
				w.handler.OnUpdate(*alloc)
				continue
			}

			w.handler.OnAdd(*alloc)
		}
	}

	w.logger.Debugf("Allocations index has changed remoteWaitIndex=%v localWaitIndex=%v",
		fmt.Sprint(remoteWaitIndex), fmt.Sprint(localWaitIndex))

	w.waitIndex = meta.LastIndex

	return nil
}

func (w *watcher) watch() {
	// Failures counter, do exponential backoff on retries
	var failures uint
	logp.Info("Nomad: %s", "Watching API for resource events")
	ticker := time.NewTicker(w.options.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return
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
