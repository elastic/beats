// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"errors"

	"github.com/elastic/entcollect"

	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/es"
)

var _ entcollect.Registry = (*elasticStorage)(nil)

// Store returns an [entcollect.Store] backed by Elasticsearch. The
// returned store is scoped to name, which determines the ES index.
func (e *elasticStorage) Store(name string) (entcollect.Store, error) {
	return &entcollectStore{base: es.NewStore(e.ctx, e.logger, e.client, name)}, nil
}

// entcollectStore wraps a [backend.Store] as [entcollect.Store],
// translating method names and error sentinels.
type entcollectStore struct {
	base backend.Store
}

func (s *entcollectStore) Get(key string, dst any) error {
	err := s.base.Get(key, dst)
	if errors.Is(err, es.ErrKeyUnknown) {
		return entcollect.ErrKeyNotFound
	}
	return err
}

func (s *entcollectStore) Set(key string, value any) error {
	return s.base.Set(key, value)
}

// Delete removes a key from the store. The entcollect.Store contract
// requires that deleting an absent key is not an error, but the
// underlying ES store returns an error on 404. We check Has first
// to avoid that. The extra round trip is acceptable: Delete is
// called rarely (IDSet shard cleanup on rehash) and only one
// goroutine accesses the store.
func (s *entcollectStore) Delete(key string) error {
	ok, err := s.base.Has(key)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return s.base.Remove(key)
}

func (s *entcollectStore) Each(fn func(key string, decode func(any) error) (bool, error)) error {
	return s.base.Each(func(key string, dec backend.ValueDecoder) (bool, error) {
		return fn(key, dec.Decode)
	})
}
