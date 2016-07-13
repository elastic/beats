package beat

import (
	"io"
	"sort"
	"sync"

	"github.com/elastic/beats/libbeat/logp"
)

type Done struct {
	C      <-chan struct{}
	OnStop StopHandler
}

// stop handle used to unregister StopHandler callbacks
type StopHandle uint64

type StopHandler interface {
	Close(io.Closer) StopHandle
	Stop(Stopper) StopHandle
	Exec(fn func()) StopHandle

	Remove(StopHandle)
}

type Stopper interface {
	Stop()
}

type beatStopHandler struct {
	mutex     sync.Mutex
	idx       StopHandle
	callbacks map[StopHandle]func()
}

type uint64Sorter struct {
	arr []uint64
}

func (d Done) Finished() bool {
	select {
	case <-d.C:
		return true
	default:
		return false
	}
}

func (d Done) Wait() { <-d.C }

func (d Done) Loop(fn func() error) error {
	for !d.Finished() {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

func (b *beatStopHandler) signal() {
	callbacks := b.collectCallbacks()

	// run callbacks in reverse order
	for i := len(callbacks) - 1; i >= 0; i-- {
		callbacks[i]()
	}
}

func (b *beatStopHandler) collectCallbacks() []func() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	keys := make([]uint64, 0, len(b.callbacks))
	for k, _ := range b.callbacks {
		keys = append(keys, uint64(k))
	}

	sorter := &uint64Sorter{keys}
	sort.Sort(sorter)
	callbacks := make([]func(), len(keys))
	for i, key := range keys {
		callbacks[i] = b.callbacks[StopHandle(key)]
	}
	return callbacks
}

func (b *beatStopHandler) Close(c io.Closer) StopHandle {
	return b.Exec(func() {
		err := c.Close()
		if err != nil {
			logp.Info("Close error: %v", err)
		}
	})
}

func (b *beatStopHandler) Stop(s Stopper) StopHandle {
	return b.Exec(s.Stop)
}

func (b *beatStopHandler) Exec(cb func()) StopHandle {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// TODO: check handle is still free in case of idx overflow
	handle := b.idx
	b.idx++
	b.callbacks[handle] = cb
	return handle
}

func (b *beatStopHandler) Remove(h StopHandle) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	delete(b.callbacks, h)
}

func (u *uint64Sorter) Len() int {
	return len(u.arr)
}

func (u *uint64Sorter) Less(i, j int) bool {
	return u.arr[i] < u.arr[j]
}

func (u *uint64Sorter) Swap(i, j int) {
	u.arr[i], u.arr[j] = u.arr[j], u.arr[i]
}
