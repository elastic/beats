// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchclient

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
)

const componentType = "elasticsearchclient"

var (
	// Type is the OTel component type for this extension.
	Type               = component.MustNewType(componentType)
	ExtensionStability = component.StabilityLevelStable
)

// NewFactory creates the elasticsearchclient extension factory.
func NewFactory() extension.Factory {
	return extension.NewFactory(
		Type,
		createDefaultConfig,
		createExtension,
		ExtensionStability,
	)
}

func createExtension(
	_ context.Context,
	set extension.Settings,
	cfg component.Config,
) (extension.Extension, error) {

	config, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("could not convert otel config to elasticsearchclient config")
	}
	logger, err := logp.NewZapLogger(set.Logger.Named(componentType))
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return &elasticsearchClient{
		cfg:    config,
		logger: logger,
		info: beat.Info{
			Beat:   componentType,
			Logger: logger,
		},
	}, nil
}
