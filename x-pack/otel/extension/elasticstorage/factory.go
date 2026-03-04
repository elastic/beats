// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticstorage

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/otel/extension/elasticstorage/internal/metadata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

func NewFactory() extension.Factory {
	return extension.NewFactory(metadata.Type, createDefaultConfig, newExtension, component.StabilityLevelDevelopment)
}

func newExtension(ctx context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	return &elasticStorage{cfg: cfg.(*Config), logger: set.Logger.Named("elasticstorage")}, nil
}
