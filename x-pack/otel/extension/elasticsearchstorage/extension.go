// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"context"

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/es"
	cfg "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

var _ extension.Extension = (*elasticStorage)(nil)
var _ backend.Registry = (*elasticStorage)(nil)

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
	// no-op. Client will be close in Shutdown
	return nil
}
