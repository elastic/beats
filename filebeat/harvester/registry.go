package harvester

import (
	"errors"
	"sync"

	uuid "github.com/satori/go.uuid"

	"github.com/elastic/beats/libbeat/logp"
)

// Registry struct manages (start / stop) a list of harvesters
type Registry struct {
	sync.RWMutex
	harvesters map[uuid.UUID]Harvester
	wg         sync.WaitGroup
	done       chan struct{}
}

// NewRegistry creates a new registry object
func NewRegistry() *Registry {
	return &Registry{
		harvesters: map[uuid.UUID]Harvester{},
		done:       make(chan struct{}),
	}
}

func (r *Registry) remove(h Harvester) {
	r.Lock()
	defer r.Unlock()
	delete(r.harvesters, h.ID())
}

func (r *Registry) add(h Harvester) {
	r.Lock()
	defer r.Unlock()
	r.harvesters[h.ID()] = h
}

// Stop stops all harvesters in the registry
func (r *Registry) Stop() {
	r.Lock()
	defer func() {
		r.Unlock()
		r.WaitForCompletion()
	}()
	// Makes sure no new harvesters are added during stopping
	close(r.done)

	for _, hv := range r.harvesters {
		go func(h Harvester) {
			h.Stop()
		}(hv)
	}
}

// WaitForCompletion can be used to wait until all harvesters are stopped
func (r *Registry) WaitForCompletion() {
	r.wg.Wait()
}

// Start starts the given harvester and add its to the registry
func (r *Registry) Start(h Harvester) error {
	// Make sure stop is not called during starting a harvester
	r.Lock()
	defer r.Unlock()

	// Make sure no new harvesters are started after stop was called
	if !r.active() {
		return errors.New("registry already stopped")
	}

	r.wg.Add(1)
	go func() {
		defer func() {
			r.remove(h)
			r.wg.Done()
		}()
		r.add(h)
		// Starts harvester and picks the right type. In case type is not set, set it to default (log)
		err := h.Run()
		if err != nil {
			logp.Err("Error running input: %v", err)
		}
	}()
	return nil
}

// Len returns the current number of harvesters in the registry
func (r *Registry) Len() uint64 {
	r.RLock()
	defer r.RUnlock()
	return uint64(len(r.harvesters))
}

func (r *Registry) active() bool {
	select {
	case <-r.done:
		return false
	default:
		return true
	}
}
