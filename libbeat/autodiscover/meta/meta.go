package meta

import (
	"sync"

	"github.com/elastic/beats/libbeat/common"
)

// Map stores a map of id -> MapStrPointer
type Map struct {
	mutex sync.RWMutex
	meta  map[uint64]common.MapStrPointer
}

// NewMap instantiates and returns a new meta.Map
func NewMap() *Map {
	return &Map{
		meta: make(map[uint64]common.MapStrPointer),
	}
}

// Store inserts or updates given meta under the given id. Then it returns a MapStrPointer to it
func (m *Map) Store(id uint64, meta common.MapStr) common.MapStrPointer {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if meta == nil {
		return common.MapStrPointer{}
	}

	p, ok := m.meta[id]
	if !ok {
		// create
		p = common.NewMapStrPointer(meta)
		m.meta[id] = p
	} else {
		// update
		p.Set(meta)
	}

	return p
}

// Remove meta stored under the given id
func (m *Map) Remove(id uint64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.meta, id)
}
