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
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

var (
	ErrClosed           = errors.New("attempt to use a closed processor")
	ErrPathsAlreadySet  = errors.New("attempt to set paths twice")
	ErrSetPathsOnClosed = errors.New("attempt to set paths on closed processor")
)

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
}

// safeProcessorWithClose extends SafeProcessor to also handle Close.
// It ensures Close is called only once on the underlying processor.
type safeProcessorWithClose struct {
	SafeProcessor
}

// Run delegates to the underlying processor. Returns ErrClosed if the processor
// has been closed.
func (p *SafeProcessor) Run(event *beat.Event) (*beat.Event, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.state == stateClosed {
		return nil, ErrClosed
	}
	return p.Processor.Run(event)
}

// Close makes sure the underlying `Close` function is called only once.
func (p *safeProcessorWithClose) Close() (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state != stateClosed {
		p.state = stateClosed
		return Close(p.Processor)
	}
	logp.L().Warnf("tried to close already closed %q processor", p.String())
	return nil
}

// SetPaths delegates to the underlying processor if it implements SetPather.
// Returns ErrPathsAlreadySet if called more than once, or ErrSetPathsOnClosed
// if the processor has been closed.
func (p *SafeProcessor) SetPaths(paths *paths.Path) error {
	setPather, ok := p.Processor.(SetPather)
	if !ok {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.state {
	case stateInit:
		p.state = stateSetPaths
		return setPather.SetPaths(paths)
	case stateSetPaths:
		return ErrPathsAlreadySet
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
func SafeWrap(constructor Constructor) Constructor {
	return func(config *config.C, log *logp.Logger) (beat.Processor, error) {
		processor, err := constructor(config, log)
		if err != nil {
			return nil, err
		}
		// if the processor does not implement `Closer` it does not need a wrap
		if _, ok := processor.(Closer); !ok {
			// if SetPaths is implemented, ensure single call of SetPaths
			if _, ok = processor.(SetPather); ok {
				return &SafeProcessor{Processor: processor}, nil
			}
			return processor, nil
		}

		return &safeProcessorWithClose{
			SafeProcessor: SafeProcessor{Processor: processor},
		}, nil
	}
}
