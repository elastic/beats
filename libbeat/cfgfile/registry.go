package cfgfile

import "sync"

type Registry struct {
	sync.Mutex
	List map[uint64]Runner
}

func NewRegistry() *Registry {
	return &Registry{
		List: map[uint64]Runner{},
	}
}

func (r *Registry) Add(hash uint64, m Runner) {
	r.Lock()
	defer r.Unlock()
	r.List[hash] = m
}

func (r *Registry) Remove(hash uint64) {
	r.Lock()
	defer r.Unlock()
	delete(r.List, hash)
}

func (r *Registry) Has(hash uint64) bool {
	r.Lock()
	defer r.Unlock()

	_, ok := r.List[hash]
	return ok
}

func (r *Registry) CopyList() map[uint64]Runner {
	r.Lock()
	defer r.Unlock()

	// Create a copy of the list
	list := map[uint64]Runner{}
	for k, v := range r.List {
		list[k] = v
	}
	return list
}
