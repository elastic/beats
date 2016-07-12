package beat

import (
	"io"
	"sync"

	"github.com/elastic/beats/libbeat/logp"
)

type Done struct {
	C      <-chan struct{}
	OnStop StopHandler
}

type StopHandler interface {
	Close(io.Closer)
	Stop(Stopper)
	Exec(fn func())
}

type Stopper interface {
	Stop()
}

type beatStopHandler struct {
	mutex     sync.Mutex
	callbacks []func()
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
	b.mutex.Lock()
	callbacks := b.callbacks
	b.mutex.Unlock()

	// run callbacks in reverse order
	for i := len(callbacks) - 1; i >= 0; i-- {
		callbacks[i]()
	}
}

func (b *beatStopHandler) Close(c io.Closer) {
	b.Exec(func() {
		err := c.Close()
		if err != nil {
			logp.Info("Close error: %v", err)
		}
	})
}

func (b *beatStopHandler) Stop(s Stopper) {
	b.Exec(s.Stop)
}

func (b *beatStopHandler) Exec(cb func()) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.callbacks = append(b.callbacks)
}
