package common

import (
	"sync/atomic"
	"unsafe"
)

// MapStrPointer stores a pointer to atomically get/set a MapStr object
// This should give faster access for use cases with lots of reads and a few
// changes.
// It's imortant to note that modifying the map is not thread safe, only fully
// replacing it.
type MapStrPointer struct {
	p *unsafe.Pointer
}

// NewMapStrPointer initializes and returns a pointer to the given MapStr
func NewMapStrPointer(m MapStr) MapStrPointer {
	pointer := unsafe.Pointer(&m)
	return MapStrPointer{p: &pointer}
}

// Get returns the MapStr stored under this pointer
func (m MapStrPointer) Get() MapStr {
	if m.p == nil {
		return nil
	}
	return *(*MapStr)(atomic.LoadPointer(m.p))
}

// Set stores a pointer the given MapStr, replacing any previous one
func (m *MapStrPointer) Set(p MapStr) {
	atomic.StorePointer(m.p, unsafe.Pointer(&p))
}
