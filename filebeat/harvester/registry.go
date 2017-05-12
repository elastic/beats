package harvester

import (
	"sync"

	uuid "github.com/satori/go.uuid"
)

// Registry struct manages (start / stop) a list of harvesters
type Registry struct {
	sync.RWMutex
	harvesters map[uuid.UUID]Harvester
	wg         sync.WaitGroup
}

// NewRegistry creates a new registry object
func NewRegistry() *Registry {
	return &Registry{
		harvesters: map[uuid.UUID]Harvester{},
	}
}

func (hr *Registry) add(h Harvester) {
	hr.Lock()
	defer hr.Unlock()
	hr.harvesters[h.ID()] = h
}

func (hr *Registry) remove(h Harvester) {
	hr.Lock()
	defer hr.Unlock()
	delete(hr.harvesters, h.ID())
}

// Stop stops all harvesters in the registry
func (hr *Registry) Stop() {
	hr.Lock()
	for _, hv := range hr.harvesters {
		hr.wg.Add(1)
		go func(h Harvester) {
			hr.wg.Done()
			h.Stop()
		}(hv)
	}
	hr.Unlock()
	hr.WaitForCompletion()
}

// WaitForCompletion can be used to wait until all harvesters are stopped
func (hr *Registry) WaitForCompletion() {
	hr.wg.Wait()
}

// Start starts the given harvester and add its to the registry
func (hr *Registry) Start(h Harvester) {

	hr.wg.Add(1)
	hr.add(h)

	go func() {
		defer func() {
			hr.remove(h)
			hr.wg.Done()
		}()
		// Starts harvester and picks the right type. In case type is not set, set it to default (log)
		h.Start()
	}()
}

// Len returns the current number of harvesters in the registry
func (hr *Registry) Len() uint64 {
	hr.RLock()
	defer hr.RUnlock()
	return uint64(len(hr.harvesters))
}
