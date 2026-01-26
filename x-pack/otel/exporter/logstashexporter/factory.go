// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstashexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
)

var (
	Type              = component.MustNewType("logstash")
	LogStabilityLevel = component.StabilityLevelDevelopment
)

func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		Type,
		createDefaultConfig,
		exporter.WithLogs(createLogExporter, LogStabilityLevel),
	)
}

func createLogExporter(_ context.Context, settings exporter.Settings, cfg component.Config) (exporter.Logs, error) {
	return newLogstashExporter(settings, cfg)
}
