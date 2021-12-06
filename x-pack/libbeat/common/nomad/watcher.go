// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package nomad

import (
	"fmt"
	"time"

	api "github.com/hashicorp/nomad/api"

	"github.com/elastic/beats/v7/libbeat/logp"
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
	// Namespace is used for filtering events on specified namespaces.
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

	if options.Node != "" {
		nodeID, err := w.fetchNodeID()
		if err != nil {
			return nil, err
		}
		w.nodeID = nodeID
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
	w.logger.Debug("Syncing allocations and metadata")
	w.logger.Debugf("Starting with WaitIndex=%v", w.waitIndex)

	queryOpts := &api.QueryOptions{
		WaitTime:   w.options.SyncTimeout,
		AllowStale: w.options.AllowStale,
		WaitIndex:  w.waitIndex,
	}

	allocations, meta, err := w.getAllocations(queryOpts)
	if err != nil {
		return fmt.Errorf("failed listing allocations: %w", err)
	}

	remoteWaitIndex := meta.LastIndex
	localWaitIndex := queryOpts.WaitIndex

	// Only emit updated metadata if the WaitIndex have changed
	if remoteWaitIndex <= localWaitIndex {
		w.logger.Debugf("Allocations index is unchanged remoteWaitIndex=%v == localWaitIndex=%v",
			remoteWaitIndex, localWaitIndex)
		return nil
	}

	w.logger.Debugf("Found %d allocations", len(allocations))
	for _, alloc := range allocations {
		// the allocation has not changed since last seen, ignore
		if w.waitIndex > alloc.ModifyIndex {
			w.logger.Debugf(
				"Skip allocation.id=%s ClientStatus=%s because w.waitIndex=%v > alloc.ModifyIndex=%v",
				alloc.ID,
				alloc.ClientStatus,
				fmt.Sprint(w.waitIndex),
				fmt.Sprint(alloc.ModifyIndex))
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

		w.logger.Debugf("Received allocation:%s DesiredStatus:%s ClientStatus:%s", alloc.ID,
			alloc.DesiredStatus, alloc.ClientStatus)

		switch alloc.ClientStatus {
		case AllocClientStatusPending:
			continue
		case AllocClientStatusComplete, AllocClientStatusFailed, AllocClientStatusLost:
			// the allocation is in a terminal state
			w.handler.OnDelete(*alloc)
		case AllocClientStatusRunning:
			// Handle in-place allocation updates (like adding tags to a service definition) that
			// don't trigger a new allocation
			updated := (w.waitIndex != 0) && (alloc.CreateIndex < w.waitIndex) && (alloc.ModifyIndex >= w.waitIndex)

			w.logger.Debugf("allocation.id=%s waitIndex=%v CreateIndex=%v ModifyIndex=%v AllocModifyIndex=%v updated=%v",
				alloc.ID, w.waitIndex, alloc.CreateIndex, alloc.ModifyIndex,
				alloc.AllocModifyIndex, updated,
			)

			if updated {
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
	w.logger.Info("Watching API for resource events")
	ticker := time.NewTicker(w.options.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			if err := w.sync(); err != nil {
				w.logger.Warnw("Error while watching for Nomad allocation changes. Backing off and continuing.", "error", err)
				backoff(failures)
				failures++
			}
		}
	}
}

func (w *watcher) getAllocations(queryOpts *api.QueryOptions) ([]*api.Allocation, *api.QueryMeta, error) {
	if w.nodeID != "" {
		return w.client.Nodes().Allocations(w.nodeID, queryOpts)
	}

	// This is making a query for each allocation in the cluster, this can be expensive,
	// consider refactoring the watcher to don't need full allocations.
	// In any case, the way to scale beats on big clusters, is to use one beat per node.
	stubs, meta, err := w.client.Allocations().List(queryOpts)
	if err != nil {
		return nil, meta, err
	}

	var allocations []*api.Allocation
	for _, stub := range stubs {
		allocation, _, err := w.client.Allocations().Info(stub.ID, queryOpts)
		if err != nil {
			w.logger.Warnw("Failed to get details of an allocation.",
				"nomad.allocation.id", stub.ID)
			continue
		}
		allocations = append(allocations, allocation)
	}
	return allocations, meta, nil
}

func (w *watcher) fetchNodeID() (string, error) {
	queryOpts := &api.QueryOptions{
		WaitTime:   w.options.SyncTimeout,
		AllowStale: w.options.AllowStale,
	}

	// Fetch the nodeId from the node name, used to filter the allocations.
	// If for some reason the NodeID changes filebeat will have to be restarted.
	nodes, _, err := w.client.Nodes().List(queryOpts)
	if err != nil {
		w.logger.Errorw("Failed fetching Nomad node list.", "error", err)
		return "", err
	}

	for _, node := range nodes {
		if node.Name == w.options.Node {
			return node.ID, nil
		}
	}

	// If there was no node with this name, check if the specified node is the local one.
	agent, err := w.client.Agent().Self()
	if err != nil {
		return "", fmt.Errorf("connecting to the nomad agent: %w", err)
	}
	if agent.Member.Name == w.options.Node {
		stats, ok := agent.Stats["client"]
		if !ok {
			return "", fmt.Errorf("getting node_id from the API client: %w", err)
		}

		nodeID, ok := stats["node_id"]
		if !ok {
			return "", fmt.Errorf("getting node_id from the API client: %w", err)
		}

		return nodeID, nil
	}

	return "", fmt.Errorf("node ID for configured node '%s' couldn't be obtained or it doesn't exist", w.options.Node)
}

func backoff(failures uint) {
	wait := 1 << failures * time.Second
	if wait > maxBackoff {
		wait = maxBackoff
	}

	time.Sleep(wait)
}
