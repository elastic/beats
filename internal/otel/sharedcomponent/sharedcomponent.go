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

// Copyright The OpenTelemetry Authors
// Adapted from go.opentelemetry.io/collector/internal/sharedcomponent
// (cannot be imported directly because it is an internal package).

// Package sharedcomponent exposes functionality for components to register
// against a shared key, such as a configuration object, in order to be reused
// across multiple pipelines that reference the same component ID.
// This is particularly useful when the component relies on an expensive shared
// resource (e.g. cloud-metadata HTTP calls, Kubernetes API connections) that
// should not be duplicated.
package sharedcomponent // import "github.com/elastic/beats/v7/internal/otel/sharedcomponent"

import (
	"container/ring"
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
)

// NewMap creates a new shared-component map.
func NewMap[K comparable, V component.Component]() *Map[K, V] {
	return &Map[K, V]{
		components: map[K]*Component[V]{},
	}
}

// Map keeps a reference to all created instances for a given shared key such
// as a component configuration pointer.
type Map[K comparable, V component.Component] struct {
	lock       sync.Mutex
	components map[K]*Component[V]
}

// LoadOrStore returns the already-created instance for key if one exists,
// otherwise calls create, stores the result, and returns it.
func (m *Map[K, V]) LoadOrStore(key K, create func() (V, error)) (*Component[V], error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if c, ok := m.components[key]; ok {
		return c, nil
	}
	comp, err := create()
	if err != nil {
		return nil, err
	}
	newComp := &Component[V]{
		component: comp,
		removeFunc: func() {
			m.lock.Lock()
			defer m.lock.Unlock()
			delete(m.components, key)
		},
	}
	m.components[key] = newComp
	return newComp, nil
}

// Len returns the number of components currently held in the map.
func (m *Map[K, V]) Len() int {
	m.lock.Lock()
	defer m.lock.Unlock()
	return len(m.components)
}

// Component ensures that the wrapped component is started and stopped only
// once. When stopped it is removed from the Map.
type Component[V component.Component] struct {
	component V

	startOnce  sync.Once
	stopOnce   sync.Once
	removeFunc func()

	hostWrapper *hostWrapper
}

// Unwrap returns the original component.
func (c *Component[V]) Unwrap() V {
	return c.component
}

// Start starts the underlying component if it has never been started before.
// Subsequent calls are no-ops for the underlying component but register
// additional status reporters.
func (c *Component[V]) Start(ctx context.Context, host component.Host) error {
	if c.hostWrapper == nil {
		var err error
		c.startOnce.Do(func() {
			c.hostWrapper = &hostWrapper{
				host:           host,
				sources:        make([]componentstatus.Reporter, 0),
				previousEvents: ring.New(5),
			}
			if statusReporter, ok := host.(componentstatus.Reporter); ok {
				c.hostWrapper.addSource(statusReporter)
			}
			c.hostWrapper.Report(componentstatus.NewEvent(componentstatus.StatusStarting))
			if err = c.component.Start(ctx, c.hostWrapper); err != nil {
				c.hostWrapper.Report(componentstatus.NewPermanentErrorEvent(err))
			}
		})
		return err
	}
	if statusReporter, ok := host.(componentstatus.Reporter); ok {
		c.hostWrapper.addSource(statusReporter)
	}
	return nil
}

// Shutdown shuts down the underlying component exactly once, then removes it
// from the parent Map so the same configuration can be recreated if needed.
func (c *Component[V]) Shutdown(ctx context.Context) error {
	var err error
	c.stopOnce.Do(func() {
		if c.hostWrapper != nil {
			c.hostWrapper.Report(componentstatus.NewEvent(componentstatus.StatusStopping))
		}
		err = c.component.Shutdown(ctx)
		if c.hostWrapper != nil {
			if err != nil {
				c.hostWrapper.Report(componentstatus.NewPermanentErrorEvent(err))
			} else {
				c.hostWrapper.Report(componentstatus.NewEvent(componentstatus.StatusStopped))
			}
		}
		c.removeFunc()
	})
	return err
}

var (
	_ component.Host           = (*hostWrapper)(nil)
	_ componentstatus.Reporter = (*hostWrapper)(nil)
)

type hostWrapper struct {
	host           component.Host
	sources        []componentstatus.Reporter
	previousEvents *ring.Ring
	lock           sync.Mutex
}

func (h *hostWrapper) GetExtensions() map[component.ID]component.Component {
	return h.host.GetExtensions()
}

func (h *hostWrapper) Report(e *componentstatus.Event) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if len(h.sources) > 0 {
		h.previousEvents.Value = e
		h.previousEvents = h.previousEvents.Next()
	}
	for _, s := range h.sources {
		s.Report(e)
	}
}

func (h *hostWrapper) addSource(s componentstatus.Reporter) {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.previousEvents.Do(func(a any) {
		if e, ok := a.(*componentstatus.Event); ok {
			s.Report(e)
		}
	})
	h.sources = append(h.sources, s)
}
