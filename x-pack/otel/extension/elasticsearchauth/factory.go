// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchauth

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

var (
	// Type is the Elasticsearch authentication extension component type.
	Type = component.MustNewType("elasticsearchauth")
	// ExtensionStability is the initial stability for this extension.
	ExtensionStability = component.StabilityLevelDevelopment
)

// NewFactory creates the Elasticsearch authentication extension factory.
func NewFactory() extension.Factory {
	return extension.NewFactory(Type, createDefaultConfig, createExtension, ExtensionStability)
}

func createExtension(ctx context.Context, _ extension.Settings, cfg component.Config) (extension.Extension, error) {
	config, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid elasticsearchauth configuration type %T", cfg)
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	tlsConfig, err := config.ClientConfig.TLS.LoadTLSConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("invalid Elasticsearch TLS configuration: %w", err)
	}
	return newAuthenticator(config, tlsConfig), nil
}
