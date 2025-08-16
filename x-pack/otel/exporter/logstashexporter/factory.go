// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstashexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
)

var (
	Name              = "logstash"
	LogStabilityLevel = component.StabilityLevelDevelopment
)

func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		component.MustNewType(Name),
		createDefaultConfig,
		exporter.WithLogs(createLogsExporter, LogStabilityLevel),
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createLogsExporter(
	ctx context.Context,
	settings exporter.Settings,
	config component.Config,
) (exporter.Logs, error) {
	return exporterhelper.NewLogs(
		ctx,
		settings,
		config,
		pushLogData,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
	)
}

func pushLogData(context.Context, plog.Logs) error {
	return nil
}
