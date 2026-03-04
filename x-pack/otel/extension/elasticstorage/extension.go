package elasticstorage

import (
	"context"

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	cfg "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ extension.Extension = (*elasticStorage)(nil)
var _ backend.Registry = (*elasticStorage)(nil)

type elasticStorage struct {
	cfg    *Config
	logger *zap.Logger
	client *eslegclient.Connection
}

func (e *elasticStorage) Start(ctx context.Context, host component.Host) error {
	c, err := cfg.NewConfigFrom(e.cfg.ElasticsearchConfig)
	if err != nil {
		return err
	}
	client, err := eslegclient.NewConnectedClient(ctx, c, "Filebeat", logp.NewLogger("", zap.WrapCore(func(zapcore.Core) zapcore.Core {
		return e.logger.Core()
	})))
	if err != nil {
		return err
	}
	e.client = client
	return nil
}

func (e *elasticStorage) Shutdown(ctx context.Context) error {
	return e.client.Close()
}

func (e *elasticStorage) Access(name string) (backend.Store, error) {
	return openStore(e.client, name)
}

func (e *elasticStorage) Close() error {
	return e.client.Close()
}
