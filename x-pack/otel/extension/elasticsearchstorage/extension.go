// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"context"
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/es"
	cfg "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/xextension/storage"
)

var (
	_ extension.Extension = (*elasticStorage)(nil)
	_ backend.Registry    = (*elasticStorage)(nil)
	_ backend.Store       = (*lockedStore)(nil)
	_ storage.Extension   = (*elasticStorage)(nil)
)

type elasticStorage struct {
	cfg    *Config
	ctx    context.Context
	logger *logp.Logger

	// clientMu guards client and serializes every operation that goes
	// through it. The connection (and its body encoder + response buffer) is
	// documented as not thread-safe, so all stores returned by Access must
	// share this lock and hold it for the full Marshal → execRequest →
	// HTTP.Do path. Shutdown sets client to nil under this lock.
	clientMu sync.Mutex
	client   *eslegclient.Connection
}

func (e *elasticStorage) Start(ctx context.Context, host component.Host) error {
	c, err := cfg.NewConfigFrom(e.cfg.ElasticsearchConfig)
	if err != nil {
		return err
	}
	client, err := eslegclient.NewConnectedClient(ctx, c, beat.Info{Beat: "Filebeat", Logger: e.logger})
	if err != nil {
		return err
	}
	e.clientMu.Lock()
	e.client = client
	e.clientMu.Unlock()
	e.ctx = ctx
	return nil
}

// Shutdown closes the shared connection and sets client to nil so a
// GetClient or Access after Shutdown fails instead of handing out a store
// backed by a closed connection.
func (e *elasticStorage) Shutdown(ctx context.Context) error {
	e.clientMu.Lock()
	defer e.clientMu.Unlock()
	if e.client == nil {
		return nil
	}
	err := e.client.Close()
	e.client = nil
	return err
}

func (e *elasticStorage) Access(name string) (backend.Store, error) {
	e.clientMu.Lock()
	defer e.clientMu.Unlock()
	if e.client == nil {
		return nil, fmt.Errorf("elasticsearch_storage: Access called before Start or after Shutdown")
	}
	return &lockedStore{
		inner: es.NewStore(e.ctx, e.logger, e.client, name),
		mu:    &e.clientMu,
	}, nil
}

// GetClient implements storage.Extension: it hands an OTel component a
// storage.Client scoped to a per-identity Elasticsearch index. The index is
// created lazily on the first write, so a transient ES outage while a
// receiver acquires its client does not stop the receiver from starting.
// All clients share the extension's single connection and clientMu.
func (e *elasticStorage) GetClient(ctx context.Context, kind component.Kind, id component.ID, storageName string) (storage.Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	e.clientMu.Lock()
	defer e.clientMu.Unlock()
	if e.client == nil {
		return nil, fmt.Errorf("elasticsearch_storage: GetClient called before Start or after Shutdown")
	}
	return &esStorageClient{
		ext:      e,
		index:    composeIndexName(kind, id, storageName),
		pageSize: defaultPageSize,
	}, nil
}

// isServerless reports whether the connected cluster is Elastic Cloud
// Serverless, reading the shared connection under clientMu. The answer is
// cached by the connection after the first call.
func (e *elasticStorage) isServerless() (bool, error) {
	e.clientMu.Lock()
	defer e.clientMu.Unlock()
	if e.client == nil {
		return false, errExtensionClosed
	}
	return e.client.IsServerless(), nil
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

func (s *lockedStore) Get(key string, to any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inner.Get(key, to)
}

func (s *lockedStore) Set(key string, value any) error {
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
