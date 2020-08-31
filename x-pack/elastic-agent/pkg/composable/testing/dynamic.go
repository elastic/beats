// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package testing

import (
	"context"
	"sync"
)

// DynamicState is the state of the dynamic mapping.
type DynamicState struct {
	Mapping    map[string]interface{}
	Processors []map[string]interface{}
}

// DynamicComm is used in tests for DynamicProviderComm.
type DynamicComm struct {
	context.Context

	lock     sync.Mutex
	previous map[string]DynamicState
	current  map[string]DynamicState
}

// NewDynamicComm creates a new DynamicComm.
func NewDynamicComm(ctx context.Context) *DynamicComm {
	return &DynamicComm{
		Context:  ctx,
		previous: make(map[string]DynamicState),
		current:  make(map[string]DynamicState),
	}
}

// AddOrUpdate adds or updates a current mapping.
func (t *DynamicComm) AddOrUpdate(id string, mapping map[string]interface{}, processors []map[string]interface{}) error {
	var err error
	mapping, err = CloneMap(mapping)
	if err != nil {
		return err
	}
	processors, err = CloneMapArray(processors)
	if err != nil {
		return err
	}

	t.lock.Lock()
	defer t.lock.Unlock()

	prev, ok := t.current[id]
	if ok {
		t.previous[id] = prev
	}
	t.current[id] = DynamicState{
		Mapping:    mapping,
		Processors: processors,
	}
	return nil
}

// Remove removes the a mapping.
func (t *DynamicComm) Remove(id string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	prev, ok := t.current[id]
	if ok {
		t.previous[id] = prev
	}
	delete(t.current, id)
}

// Previous returns the previous set mapping for ID.
func (t *DynamicComm) Previous(id string) (DynamicState, bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	prev, ok := t.previous[id]
	return prev, ok
}

// PreviousIDs returns the previous set mapping ID.
func (t *DynamicComm) PreviousIDs() []string {
	t.lock.Lock()
	defer t.lock.Unlock()
	var keys []string
	for key := range t.previous {
		keys = append(keys, key)
	}
	return keys
}

// Current returns the current set mapping for ID.
func (t *DynamicComm) Current(id string) (DynamicState, bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	curr, ok := t.current[id]
	return curr, ok
}

// CurrentIDs returns the current set mapping ID.
func (t *DynamicComm) CurrentIDs() []string {
	t.lock.Lock()
	defer t.lock.Unlock()
	var keys []string
	for key := range t.current {
		keys = append(keys, key)
	}
	return keys
}

// Deleted returns ture if mapping ID was deleted.
func (t *DynamicComm) Deleted(id string) bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	_, prevOk := t.previous[id]
	_, currOk := t.current[id]
	return prevOk && !currOk
}
