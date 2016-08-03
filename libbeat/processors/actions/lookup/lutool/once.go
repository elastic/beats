package lutool

import (
	"sync"
	"sync/atomic"
)

type execOnce struct {
	m    sync.Mutex
	done uint32
}

func (e *execOnce) Do(fn func() error) error {
	if atomic.LoadUint32(&e.done) == 1 {
		return nil
	}

	// Slow path. Run fn and store result only if fn did not error.
	e.m.Lock()
	defer e.m.Unlock()
	if e.done == 0 {
		err := fn()
		if err != nil {
			return err
		}
		atomic.StoreUint32(&e.done, 1)
	}

	return nil
}
