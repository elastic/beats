package nomad

import (
	"time"

	"github.com/elastic/beats/libbeat/logp"
	nomad "github.com/hashicorp/nomad/api"
)

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

func (w *watcher) Stop() {
	close(w.stopCh)
}
