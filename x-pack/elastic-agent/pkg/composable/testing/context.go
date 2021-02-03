// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package testing

import (
	"context"
	"sync"
)

// ContextComm is used in tests for ContextProviderComm.
type ContextComm struct {
	context.Context

	lock     sync.Mutex
	previous map[string]interface{}
	current  map[string]interface{}
	onSet    func()
}

// NewContextComm creates a new ContextComm.
func NewContextComm(ctx context.Context) *ContextComm {
	return &ContextComm{
		Context: ctx,
	}
}

// Set sets the current mapping for the context.
func (t *ContextComm) Set(mapping map[string]interface{}) error {
	var err error
	mapping, err = CloneMap(mapping)
	if err != nil {
		return err
	}

	t.lock.Lock()
	t.previous = t.current
	t.current = mapping
	onSet := t.onSet
	t.lock.Unlock()

	if onSet != nil {
		onSet()
	}
	return nil
}

// Previous returns the previous set mapping.
func (t *ContextComm) Previous() map[string]interface{} {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.previous
}

// Current returns the current set mapping.
func (t *ContextComm) Current() map[string]interface{} {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.current
}

// CallOnSet sets the OnSet callback.
func (t *ContextComm) CallOnSet(f func()) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.onSet = f
}
