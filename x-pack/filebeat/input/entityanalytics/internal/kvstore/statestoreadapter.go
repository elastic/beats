// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/entcollect"
)

var _ entcollect.Store = (*StateStoreAdapter)(nil)

// StateStoreAdapter wraps a *statestore.Store to satisfy the
// entcollect.Store interface. It is used when the ES-backed state
// store is enabled for agentless deployments.
//
// Callers must call store.SetID before constructing the adapter to
// ensure per-input isolation in the ES backend.
type StateStoreAdapter struct {
	store *statestore.Store
}

// NewStateStoreAdapter returns an entcollect.Store backed by s.
func NewStateStoreAdapter(s *statestore.Store) *StateStoreAdapter {
	return &StateStoreAdapter{store: s}
}

func (a *StateStoreAdapter) Get(key string, dst any) error {
	err := a.store.Get(key, dst)
	if err != nil {
		if isKeyUnknown(err) {
			return fmt.Errorf("state store get %q: %w", key, entcollect.ErrKeyNotFound)
		}
		return fmt.Errorf("state store get %q: %w", key, err)
	}
	return nil
}

func (a *StateStoreAdapter) Set(key string, value any) error {
	// entcollect.Buffer.Commit passes json.RawMessage to Set. The ES
	// backend's encoder uses struct-to-map conversion which doesn't
	// preserve json.RawMessage semantics (it treats []byte as a byte
	// array). Decode into a generic value so the encoder gets a
	// proper Go type.
	if raw, ok := value.(json.RawMessage); ok {
		var decoded any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return fmt.Errorf("state store set %q: decode raw: %w", key, err)
		}
		value = decoded
	}
	err := a.store.Set(key, value)
	if err != nil {
		return fmt.Errorf("state store set %q: %w", key, err)
	}
	return nil
}

func (a *StateStoreAdapter) Delete(key string) error {
	has, err := a.store.Has(key)
	if err != nil {
		return fmt.Errorf("state store delete check %q: %w", key, err)
	}
	if !has {
		return nil
	}
	err = a.store.Remove(key)
	if err != nil {
		return fmt.Errorf("state store delete %q: %w", key, err)
	}
	return nil
}

func (a *StateStoreAdapter) Each(fn func(key string, decode func(any) error) (bool, error)) error {
	return a.store.Each(func(key string, dec statestore.ValueDecoder) (bool, error) {
		return fn(key, dec.Decode)
	})
}

// isKeyUnknown checks whether err represents a key-not-found from
// any statestore backend. All backends (ES, memlog, otelstorage)
// use "key unknown" as the error message for missing keys.
func isKeyUnknown(err error) bool {
	var opErr *statestore.ErrorOperation
	if errors.As(err, &opErr) {
		cause := opErr.Unwrap()
		return cause != nil && cause.Error() == "key unknown"
	}
	return false
}
