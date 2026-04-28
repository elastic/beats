// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kafkapartitionerextension

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

var Type component.Type = component.MustNewType("kafkapartitioner")

func NewFactory() extension.Factory {
	return extension.NewFactory(Type, createDefaultConfig, newExtension, component.StabilityLevelDevelopment)
}

func newExtension(ctx context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	config, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("could not convert otel config to kafkapartitioner config")
	}
	return &kafkaPartitioner{cfg: config, logger: set.Logger.Named("kafkapartitioner")}, nil
}
