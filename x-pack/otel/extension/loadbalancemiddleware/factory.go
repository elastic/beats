// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package loadbalancemiddleware

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

var (
	Type               = component.MustNewType("loadbalancemiddleware")
	ExtensionStability = component.StabilityLevelDevelopment
)

func NewFactory() extension.Factory {
	return extension.NewFactory(
		Type,
		createDefaultConfig,
		createExtension,
		ExtensionStability,
	)
}

func createExtension(_ context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	return newLoadBalanceMiddleware(cfg.(*Config), set.TelemetrySettings)
}
