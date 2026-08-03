// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package processors

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

var (
	ErrClosed           = errors.New("attempt to use a closed processor")
	ErrPathsNotSet      = errors.New("attempt to run processor before SetPaths was called")
	ErrPathsAlreadySet  = errors.New("attempt to set paths twice")
	ErrSetPathsOnClosed = errors.New("attempt to set paths on closed processor")
)

var _ PdataProcessor = (*safePdataProcessorWithClose)(nil)

type state = int

const (
	stateInit state = iota
	stateSetPaths
	stateClosed
)

// SafeProcessor wraps a beat.Processor to provide thread-safe state management.
// It ensures SetPaths is called only once and prevents Run after Close.
// Use safeProcessorWithClose for processors that also implement Closer.
type SafeProcessor struct {
	beat.Processor

	mu    sync.RWMutex
	state state
	paths *paths.Path
}

// safeProcessorWithClose extends SafeProcessor to also handle Close.
// It ensures Close is called only once on the underlying processor.
type safeProcessorWithClose struct {
	SafeProcessor
}

// safePdataProcessorWithClose extends safeProcessorWithClose with a pdata fast
// path. It is only created by SafeWrap when the inner processor implements both
// Closer and PdataProcessor.
type safePdataProcessorWithClose struct {
	safeProcessorWithClose
	pdataProc PdataProcessor
}

// Run delegates to the underlying processor. Returns ErrClosed if the processor
// has been closed, or ErrPathsNotSet if the processor implements PathSetter but
// SetPaths has not been called.
func (p *SafeProcessor) Run(event *beat.Event) (*beat.Event, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	switch p.state {
	case stateClosed:
		return nil, ErrClosed
	case stateInit:
		if _, ok := p.Processor.(PathSetter); ok {
			return nil, ErrPathsNotSet
		}
	default: // proceed
	}
	return p.Processor.Run(event)
}

// SetPaths delegates to the underlying processor if it implements PathSetter.
// Returns ErrPathsAlreadySet if called more than once, or ErrSetPathsOnClosed
// if the processor has been closed.
func (p *SafeProcessor) SetPaths(paths *paths.Path) error {
	pathSetter, ok := p.Processor.(PathSetter)
	if !ok {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.state {
	case stateInit:
		p.state = stateSetPaths
		p.paths = paths
		return pathSetter.SetPaths(paths)
	case stateSetPaths:
		if p.paths != paths {
			return ErrPathsAlreadySet
		}
		return nil
	case stateClosed:
		return ErrSetPathsOnClosed
	}
	return fmt.Errorf("unknown state: %d", p.state)
}

// Close makes sure the underlying `Close` function is called only once.
func (p *safeProcessorWithClose) Close() (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state != stateClosed {
		p.state = stateClosed
		return Close(p.Processor)
	}
	return nil
}

// RunPdata delegates to the inner processor's RunPdata. The inner is guaranteed
// to implement PdataProcessor because SafeWrap only creates this type when the
// inner satisfies both Closer and PdataProcessor.
func (p *safePdataProcessorWithClose) RunPdata(body pcommon.Map) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	switch p.state {
	case stateClosed:
		return false, ErrClosed
	case stateInit:
		if _, ok := p.Processor.(PathSetter); ok {
			return false, ErrPathsNotSet
		}
	default: // proceed
	}
	return p.pdataProc.RunPdata(body)
}

// wrapUnshared preserves SafeWrap's Close and SetPaths guarantees for processor
// instances that cannot be shared.
func wrapUnshared(processor beat.Processor) beat.Processor {
	if _, ok := processor.(Closer); ok {
		if pdataProc, ok := processor.(PdataProcessor); ok {
			return &safePdataProcessorWithClose{
				safeProcessorWithClose: safeProcessorWithClose{
					SafeProcessor: SafeProcessor{Processor: processor},
				},
				pdataProc: pdataProc,
			}
		}
		return &safeProcessorWithClose{
			SafeProcessor: SafeProcessor{Processor: processor},
		}
	}
	if _, ok := processor.(PathSetter); ok {
		return &SafeProcessor{Processor: processor}
	}
	return processor
}

type sharedProcessorKey struct {
	name       string
	configHash uint64
}

// sharedCore is the process-wide reference-counted instance for one processor key.
type sharedCore struct {
	safeProcessorWithClose

	key sharedProcessorKey

	// refCount includes live and pending handles; guarded by sharedProcessorMu.
	refCount int
	// constructionErr is published before ready closes.
	constructionErr error
	// ready is closed when construction has finished, successfully or not.
	ready chan struct{}
}

// sharedHandle releases one core reference at most once.
type sharedHandle struct {
	*sharedCore
	closed atomic.Bool
}

// sharedPdataHandle preserves PdataProcessor on a shared handle.
type sharedPdataHandle struct {
	*sharedHandle
	pdata PdataProcessor
}

var sharedProcessorMu sync.Mutex
var sharedProcessors = make(map[sharedProcessorKey]*sharedCore)

// SafeWrap wraps a processor constructor to handle common edge cases:
//
//   - Multiple Close calls: Each processor might end up in multiple processor
//     groups. Every group has its own Close that calls Close on each processor,
//     leading to multiple Close calls on the same processor. Close is
//     idempotent per constructed handle.
//
//   - Multiple SetPaths calls: The wrapper ensures SetPaths is called at most once.
//
//   - Close before/during SetPaths: Prevents initialization after shutdown and
//     protects against race conditions between SetPaths and Close.
//
// Processors with the same name and configuration share one process-wide
// instance unless they implement PathSetter or Unshareable. Handles reference
// count the shared instance, which is closed after its last handle.
//
// Without SafeWrap, processors must handle these cases manually using sync.Once
// or similar mechanisms. SafeWrap is automatically applied by RegisterPlugin.
func SafeWrap(name string, constructor Constructor) Constructor {
	return func(cfg *config.C, log *logp.Logger) (beat.Processor, error) {
		configHash, err := hashProcessorConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to hash processor config: %w", err)
		}

		core, needsConstruction := reserveSharedCore(sharedProcessorKey{name: name, configHash: configHash})
		if needsConstruction {
			return core.constructOnce(constructor, cfg, log)
		}
		return core.waitForProcessor(constructor, cfg, log)
	}
}

func hashProcessorConfig(cfg *config.C) (uint64, error) {
	if cfg == nil {
		return 0, nil
	}
	return cfgfile.HashConfig(cfg)
}

// reserveSharedCore reserves one reference and reports whether the caller must construct it.
func reserveSharedCore(key sharedProcessorKey) (*sharedCore, bool) {
	sharedProcessorMu.Lock()
	defer sharedProcessorMu.Unlock()

	if core, ok := sharedProcessors[key]; ok {
		core.refCount++
		return core, false
	}

	core := &sharedCore{key: key, refCount: 1, ready: make(chan struct{})}
	sharedProcessors[key] = core
	return core, true
}

// constructOnce constructs and publishes a reserved core.
func (c *sharedCore) constructOnce(constructor Constructor, cfg *config.C, log *logp.Logger) (processor beat.Processor, err error) {
	defer func() {
		if panicValue := recover(); panicValue != nil {
			err = fmt.Errorf("%q processor constructor panicked: %v", c.key.name, panicValue)
			sharedProcessorMu.Lock()
			c.constructionErr = err
			evictLocked(c)
			sharedProcessorMu.Unlock()
		}
		close(c.ready)
	}()

	processor, err = constructor(cfg, log)
	if err == nil && processor == nil {
		err = fmt.Errorf("%q processor constructor returned a nil processor", c.key.name)
	}

	sharedProcessorMu.Lock()
	defer sharedProcessorMu.Unlock()
	if err != nil {
		c.constructionErr = err
		// Evict failed attempts so future callers can retry construction.
		evictLocked(c)
		return nil, err
	}
	c.Processor = processor
	if !shareable(processor) {
		evictLocked(c)
		return wrapUnshared(processor), nil
	}
	return newSharedHandle(c), nil
}

func (c *sharedCore) waitForProcessor(constructor Constructor, cfg *config.C, log *logp.Logger) (beat.Processor, error) {
	<-c.ready
	if c.constructionErr != nil {
		// Construction failed, constructOnce has already evicted the core.
		return nil, c.constructionErr
	}

	if shareable(c.Processor) {
		return newSharedHandle(c), nil
	}

	// The unshareable core was evicted, so this caller needs its own instance.
	processor, err := constructor(cfg, log)
	if err != nil {
		return nil, err
	}
	return wrapUnshared(processor), nil
}

// evictLocked removes core from sharedProcessors. Callers must hold
// sharedProcessorMu.
func evictLocked(core *sharedCore) {
	delete(sharedProcessors, core.key)
}

// shareable reports whether p can be reused by independent owners.
func shareable(p beat.Processor) bool {
	switch p.(type) {
	case PathSetter, Unshareable:
		return false
	default:
		return true
	}
}

// newSharedHandle returns a handle owning one reference to core. When the shared
// processor implements PdataProcessor the returned handle also forwards
// RunPdata so the pdata fast path survives sharing.
func newSharedHandle(core *sharedCore) beat.Processor {
	h := &sharedHandle{sharedCore: core}
	if pp, ok := core.Processor.(PdataProcessor); ok {
		return &sharedPdataHandle{sharedHandle: h, pdata: pp}
	}
	return h
}

func (h *sharedHandle) Run(event *beat.Event) (*beat.Event, error) {
	if h.closed.Load() {
		return nil, ErrClosed
	}
	return h.sharedCore.Run(event)
}

func (h *sharedHandle) Close() error {
	if h.closed.Swap(true) {
		return nil
	}

	sharedProcessorMu.Lock()
	h.refCount--
	if h.refCount > 0 {
		sharedProcessorMu.Unlock()
		return nil
	}
	evictLocked(h.sharedCore)
	sharedProcessorMu.Unlock()

	// Close outside the global lock so a slow processor cannot stall unrelated
	// construction or teardown.
	return h.sharedCore.Close()
}

func (h *sharedPdataHandle) RunPdata(body pcommon.Map) (bool, error) {
	if h.closed.Load() {
		return false, ErrClosed
	}
	// A live handle holds a reference, so the core cannot be released and its
	// underlying processor cannot be closed while this runs. The RLock mirrors
	// SafeProcessor.Run and guards against a concurrent Close on the core.
	c := h.sharedCore
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.state == stateClosed {
		return false, ErrClosed
	}
	return h.pdata.RunPdata(body)
}
