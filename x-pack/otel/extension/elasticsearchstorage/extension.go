// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/es"
	cfg "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/xextension/storage"
)

var _ extension.Extension = (*elasticStorage)(nil)
var _ backend.Registry = (*elasticStorage)(nil)
var _ storage.Extension = (*elasticStorage)(nil)

const esDocType = "_doc"

type elasticStorage struct {
	cfg    *Config
	ctx    context.Context
	logger *logp.Logger
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

func (e *elasticStorage) Access(name string) (backend.Store, error) {
	return es.NewStore(e.ctx, e.logger, e.client, name), nil
}

func (e *elasticStorage) Close() error {
	// no-op. Client will be closed in Shutdown
	return nil
}

func (e *elasticStorage) GetClient(ctx context.Context, kind component.Kind, id component.ID, storageName string) (storage.Client, error) {
	if e.client == nil {
		return nil, fmt.Errorf("elasticsearch storage extension not started")
	}
	name := storageName
	if name == "" {
		name = id.String()
	}
	return &esStorageClient{
		logger: e.logger,
		client: e.client,
		index:  es.RenderIndexName(name),
	}, nil
}
