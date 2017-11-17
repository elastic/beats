package cfgfile

import "sync"

// Registry holds a list of Runners mapped by their unique hashes
type Registry struct {
	sync.RWMutex
	List map[uint64]Runner
}

// NewRegistry returns a new empty Registry
func NewRegistry() *Registry {
	return &Registry{
		List: map[uint64]Runner{},
	}
}

// Add the given Runner to the list, indexed by a hash
func (r *Registry) Add(hash uint64, m Runner) {
	r.Lock()
	defer r.Unlock()
	r.List[hash] = m
}

// Remove the Runner with the given hash from the list
func (r *Registry) Remove(hash uint64) {
	r.Lock()
	defer r.Unlock()
	delete(r.List, hash)
}

// Has returns true if there is a Runner with the given hash
func (r *Registry) Has(hash uint64) bool {
	r.RLock()
	defer r.RUnlock()

	_, ok := r.List[hash]
	return ok
}

// Get returns the Runner with the given hash, or nil if it doesn't exist
func (r *Registry) Get(hash uint64) Runner {
	r.RLock()
	defer r.RUnlock()

	return r.List[hash]
}

// CopyList returns a static copy of the Runners map
func (r *Registry) CopyList() map[uint64]Runner {
	r.RLock()
	defer r.RUnlock()

	// Create a copy of the list
	list := map[uint64]Runner{}
	for k, v := range r.List {
		list[k] = v
	}
	return list
}
