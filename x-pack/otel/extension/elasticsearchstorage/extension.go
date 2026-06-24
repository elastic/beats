// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"context"
	"sync"

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/es"
	cfg "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

var (
	_ extension.Extension = (*elasticStorage)(nil)
	_ backend.Registry    = (*elasticStorage)(nil)
)

type elasticStorage struct {
	cfg    *Config
	ctx    context.Context
	logger *logp.Logger
	client *eslegclient.Connection

	// clientMu serializes every operation that goes through client. The
	// connection (and its body encoder + response buffer) is documented as
	// not thread-safe, so all stores returned by Access must share this lock
	// and hold it for the full Marshal → execRequest → HTTP.Do path.
	clientMu sync.Mutex
}

func (e *elasticStorage) Start(ctx context.Context, host component.Host) error {
	c, err := cfg.NewConfigFrom(e.cfg.ElasticsearchConfig)
	if err != nil {
		return err
	}
	client, err := eslegclient.NewConnectedClient(ctx, c, "Filebeat", e.logger)
	if err != nil {
		return err
	}
	e.client = client
	e.ctx = ctx
	return nil
}

func (e *elasticStorage) Shutdown(ctx context.Context) error {
	e.clientMu.Lock()
	defer e.clientMu.Unlock()
	if e.client == nil {
		return nil
	}
	return e.client.Close()
}

func (e *elasticStorage) Access(name string) (backend.Store, error) {
	return &lockedStore{
		inner: es.NewStore(e.ctx, e.logger, e.client, name),
		mu:    &e.clientMu,
	}, nil
}

func (e *elasticStorage) Close() error {
	// no-op. Client will be close in Shutdown
	return nil
}

// lockedStore serializes access to the underlying baseStore so that
// concurrent callers cannot race on the shared eslegclient.Connection
// (its body encoder reuses a single *bytes.Buffer, and its response
// buffer is also shared). The mutex is owned by elasticStorage so that
// every store returned by Access shares the same lock.
type lockedStore struct {
	inner backend.Store
	mu    *sync.Mutex
}

func (s *lockedStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inner.Close()
}

func (s *lockedStore) Has(key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inner.Has(key)
}

func (s *lockedStore) Get(key string, to interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inner.Get(key, to)
}

func (s *lockedStore) Set(key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inner.Set(key, value)
}

func (s *lockedStore) Remove(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inner.Remove(key)
}

func (s *lockedStore) Each(fn func(string, backend.ValueDecoder) (bool, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inner.Each(fn)
}

func (s *lockedStore) SetID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inner.SetID(id)
}
