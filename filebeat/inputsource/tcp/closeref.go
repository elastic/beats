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

package tcp

import (
	"sync"

	"github.com/pkg/errors"
)

// CloserFunc is the function called by the Closer on `Close()`.
type CloserFunc func()

// ErrClosed is returned when the Closer is closed.
var ErrClosed = errors.New("closer is closed")

// CloseRef implements a subset of the context.Context interface and it's use to synchronize
// the shutdown of multiple go-routines.
type CloseRef interface {
	Done() <-chan struct{}
	Err() error
}

// Closer implements a shutdown strategy when dealing with multiples go-routines, it creates a tree
// of Closer, when you call `Close()` on a parent the `Close()` method will be called on the current
// closer and any of the childs it may have and will remove the current node from the parent.
//
// NOTE: The `Close()` is reentrant but will propage the close only once.
type Closer struct {
	mu       sync.Mutex
	done     chan struct{}
	err      error
	parent   *Closer
	children map[*Closer]struct{}
	callback CloserFunc
}

// Close closes the closes and propagates the close to any child, on close the close callback will
// be called, this can be used for custom cleanup like closing a TCP socket.
func (c *Closer) Close() {
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return
	}

	if c.callback != nil {
		c.callback()
	}

	close(c.done)

	// propagate close to children.
	if c.children != nil {
		for child := range c.children {
			child.Close()
		}
		c.children = nil
	}

	c.err = ErrClosed
	c.mu.Unlock()

	if c.parent != nil {
		c.removeChild(c)
	}
}

// Done returns the synchronization channel, the channel will be closed if `Close()` was called on
// the current node or any parent it may have.
func (c *Closer) Done() <-chan struct{} {
	return c.done
}

// Err returns an error if the Closer was already closed.
func (c *Closer) Err() error {
	c.mu.Lock()
	err := c.err
	c.mu.Unlock()
	return err
}

func (c *Closer) removeChild(child *Closer) {
	c.mu.Lock()
	delete(c.children, child)
	c.mu.Unlock()
}

func (c *Closer) addChild(child *Closer) {
	c.mu.Lock()
	if c.children == nil {
		c.children = make(map[*Closer]struct{})
	}
	c.children[child] = struct{}{}
	c.mu.Unlock()
}

// WithCloser wraps a new closer into a child of an existing closer.
func WithCloser(parent *Closer, fn CloserFunc) *Closer {
	child := &Closer{
		done:     make(chan struct{}),
		parent:   parent,
		callback: fn,
	}
	parent.addChild(child)
	return child
}

// NewCloser creates a new Closer.
func NewCloser(fn CloserFunc) *Closer {
	return &Closer{
		done:     make(chan struct{}),
		callback: fn,
	}
}
