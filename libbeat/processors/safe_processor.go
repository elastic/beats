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

type state = int

const (
	stateInit state = iota
	stateSetPaths
	stateClosed
)

var sharedProcessorMu sync.Mutex
var sharedProcessors map[string]map[uint64]beat.Processor = make(map[string]map[uint64]beat.Processor)

// SafeProcessor wraps a beat.Processor to provide thread-safe state management.
// It ensures SetPaths is called only once and prevents Run after Close.
// Use safeProcessorWithClose for processors that also implement Closer.
type SafeProcessor struct {
	beat.Processor

	mu    sync.RWMutex
	state state
	paths *paths.Path

	refCount int
	hash     uint64
	name     string
}

// safeProcessorWithClose extends SafeProcessor to also handle Close.
// It ensures Close is called only once on the underlying processor.
type safeProcessorWithClose struct {
	SafeProcessor
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

// Close makes sure the underlying `Close` function is called only once.
func (p *safeProcessorWithClose) Close() (err error) {
	sharedProcessorMu.Lock()
	defer sharedProcessorMu.Unlock()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state == stateClosed {
		logp.L().Warnf("tried to close already closed %q processor", p.String())
		return
	}
	p.refCount--
	if p.refCount == 0 {
		p.deleteFromSharedMap()
		p.state = stateClosed
		return Close(p.Processor)
	}
	return nil
}

// NOTE: To be called while holding the sharedProcessorMu lock to ensure.
func (p *SafeProcessor) deleteFromSharedMap() {
	if _, ok := sharedProcessors[p.name]; !ok {
		return
	}
	delete(sharedProcessors[p.name], p.hash)
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

// SafeWrap wraps a processor constructor to handle common edge cases:
//
//   - Multiple Close calls: Each processor might end up in multiple processor
//     groups. Every group has its own Close that calls Close on each processor,
//     leading to multiple Close calls on the same processor.
//
//   - Multiple SetPaths calls: The wrapper ensures SetPaths is called at most once.
//
//   - Close before/during SetPaths: Prevents initialization after shutdown and
//     protects against race conditions between SetPaths and Close.
//
// Without SafeWrap, processors must handle these cases manually using sync.Once
// or similar mechanisms. SafeWrap is automatically applied by RegisterPlugin.
func SafeWrap(name string, constructor Constructor) Constructor {
	return func(cfg *config.C, log *logp.Logger) (beat.Processor, error) {
		sharedProcessorMu.Lock()
		defer sharedProcessorMu.Unlock()
		hash, err := cfgfile.HashConfig(cfg)
		if cfg == nil {
			err = nil
			hash = 0
		}
		if err != nil {
			return nil, fmt.Errorf("failed to hash processor config: %w", err)
		}
		if p, ok := sharedProcessors[name][hash]; ok {
			switch proc := p.(type) {
			case *safeProcessorWithClose:
				proc.mu.Lock()
				defer proc.mu.Unlock()
				proc.refCount++
				return proc, nil
			case *SafeProcessor:
				proc.mu.Lock()
				defer proc.mu.Unlock()
				proc.refCount++
				return proc, nil
			}
			return p, nil
		}
		safeProcessor, err := newSafeProcessor(log, constructor, cfg, hash, name)
		if err != nil {
			return nil, err
		}
		if sharedProcessors[name] == nil {
			sharedProcessors[name] = make(map[uint64]beat.Processor)
		}
		sharedProcessors[name][hash] = safeProcessor
		return safeProcessor, nil
	}
}

func newSafeProcessor(log *logp.Logger, constructor Constructor, config *config.C, hash uint64, name string) (beat.Processor, error) {
	processor, err := constructor(config, log)
	if err != nil {
		return nil, err
	}
	// if the processor does not implement `Closer` it does not need a wrap
	if _, ok := processor.(Closer); !ok {
		// if SetPaths is implemented, ensure single call of SetPaths
		if _, ok = processor.(PathSetter); ok {
			return &SafeProcessor{Processor: processor, hash: hash, name: name, refCount: 1}, nil
		}
		return processor, nil
	}

	return &safeProcessorWithClose{
		SafeProcessor: SafeProcessor{Processor: processor, hash: hash, name: name, refCount: 1},
	}, nil
}
