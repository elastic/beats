package shared

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type SharedProcessorWithClose struct {
	beat.Processor
	hash     uint64
	refCount int
	refLock  sync.RWMutex

	sharedProcessors  map[uint64]*SharedProcessorWithClose
	sharedProcessorMu *sync.Mutex
}

// New wraps a processor constructor to return a shared processor.
// The shared processor will be shared across all processors with the same configuration.
// The shared processor will be closed when the last processor using it is closed.
func New(constructor processors.Constructor) processors.Constructor {
	sharedProcessors := make(map[uint64]*SharedProcessorWithClose)
	sharedProcessorMu := &sync.Mutex{}

	return func(cfg *config.C, logger *logp.Logger) (beat.Processor, error) {
		sharedProcessorMu.Lock()
		defer sharedProcessorMu.Unlock()

		hash, err := cfgfile.HashConfig(cfg)
		if cfg == nil {
			err = nil
			hash = 0
		}
		if err != nil {
			return nil, err
		}

		if p, ok := sharedProcessors[hash]; ok {
			p.refLock.Lock()
			p.refCount++
			p.refLock.Unlock()
			return p, nil
		}

		proc, err := constructor(cfg, logger)
		if err != nil {
			return nil, err
		}
		// if the processor does not implement `Closer` it does not need a wrap.
		// We can extend this in future, if needed.
		if _, ok := proc.(processors.Closer); !ok {
			return proc, nil
		}

		sharedProcessors[hash] = &SharedProcessorWithClose{Processor: proc, hash: hash, sharedProcessors: sharedProcessors, sharedProcessorMu: sharedProcessorMu, refCount: 1}
		return sharedProcessors[hash], nil
	}
}

func (p *SharedProcessorWithClose) Close() error {
	p.sharedProcessorMu.Lock()
	defer p.sharedProcessorMu.Unlock()
	p.refLock.Lock()
	defer p.refLock.Unlock()
	if p.refCount < 0 {
		return nil
	}
	p.refCount--
	if p.refCount == 0 {
		p.deleteFromSharedMap()
		return processors.Close(p.Processor)
	}
	return nil
}

func (p *SharedProcessorWithClose) deleteFromSharedMap() {
	if _, ok := p.sharedProcessors[p.hash]; !ok {
		return
	}
	delete(p.sharedProcessors, p.hash)
}
