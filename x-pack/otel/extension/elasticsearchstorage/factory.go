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

	"github.com/elastic/elastic-agent-libs/logp"
)

func NewFactory() extension.Factory {
	return extension.NewFactory(component.MustNewType("elasticsearch_storage"), createDefaultConfig, newExtension, component.StabilityLevelDevelopment)
}

func newExtension(ctx context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	config, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("could not convert OTel config to elasticsearch_storage config")
	}

	logger, err := logp.NewZapLogger(set.Logger.WithOptions(zap.AddCallerSkip(1)))
	if err != nil {
		return nil, fmt.Errorf("cannot create logger: %w", err)
	}

	return &elasticStorage{cfg: config, logger: logger}, nil
}
