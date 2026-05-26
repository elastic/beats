// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package elasticsearchstorage

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/xextension/storage"

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/es"
	cfg "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

var _ extension.Extension = (*elasticStorage)(nil)
var _ backend.Registry = (*elasticStorage)(nil)
var _ storage.Extension = (*elasticStorage)(nil)

type elasticStorage struct {
	cfg    *Config
	ctx    context.Context
	logger *logp.Logger

	// mu serializes all use of the shared client. eslegclient.Connection
	// reuses an internal response buffer and is documented as not safe for
	// concurrent use; holding mu around every Connection.Request call lets
	// one connection serve all consumers (legacy backend.Registry + OTel
	// storage.Client) without races.
	mu     sync.Mutex
	client *eslegclient.Connection
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
	if e.client == nil {
		return nil
	}
	return e.client.Close()
}

// Access implements backend.Registry. Returns a baseStore tied to the
// shared connection — unchanged from the previous behaviour. Existing
// Beats inputs that depend on this path continue to work.
func (e *elasticStorage) Access(name string) (backend.Store, error) {
	return es.NewStore(e.ctx, e.logger, e.client, name), nil
}

// Close implements backend.Registry. The actual *eslegclient.Connection is
// closed in Shutdown — this is a no-op for the legacy interface.
func (e *elasticStorage) Close() error {
	return nil
}

// GetClient implements storage.Extension. Returns a thin client tied to
// the shared connection (via the extension's mutex) and a per-consumer
// index whose name is composed from kind, component ID, and storageName
// and sanitized for ES naming rules. The new client path is independent
// of the legacy Access() path.
func (e *elasticStorage) GetClient(
	ctx context.Context,
	kind component.Kind,
	id component.ID,
	storageName string,
) (storage.Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e.client == nil {
		return nil, fmt.Errorf("elasticsearch_storage: GetClient called before Start")
	}

	indexName := composeIndexName(kind, id, storageName)
	if err := ensureIndex(&e.mu, e.client, indexName); err != nil {
		return nil, err
	}
	return &esStorageClient{
		ext:   e,
		index: indexName,
	}, nil
}
