// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package storage

import "io"

type handlerFunc func(io.Reader) error

// HandlerStore take a function handler and wrap it into the store interface.
type HandlerStore struct {
	fn handlerFunc
}

// NewHandlerStore takes a function and wrap it into an handlerStore.
func NewHandlerStore(fn handlerFunc) *HandlerStore {
	return &HandlerStore{fn: fn}
}

// Save calls the handler.
func (h *HandlerStore) Save(in io.Reader) error {
	return h.fn(in)
}
