// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/elastic-agent-libs/logp"
)

func NewFactory() extension.Factory {
	return extension.NewFactory(component.MustNewType("elasticsearch_storage"), createDefaultConfig, newExtension, component.StabilityLevelDevelopment)
}

func newExtension(ctx context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	config, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("could not convert otel config to elasticstorage config")
	}
	logger := logp.NewLogger("", zap.WrapCore(func(zapcore.Core) zapcore.Core {
		return set.Logger.Named("elasticstorage").Core()
	}))
	return &elasticStorage{cfg: config, logger: logger}, nil
}
