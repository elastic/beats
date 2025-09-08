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

	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/logstash"
	"github.com/elastic/elastic-agent-libs/config"
)

var (
	Name              = "logstash"
	LogStabilityLevel = component.StabilityLevelDevelopment
)

type logstashOutputConfig struct {
	outputs.HostWorkerCfg `config:",inline"`
	logstash.Config       `config:",inline"`
}

type logstashExporter struct {
	config *logstashOutputConfig
}

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
	cfg component.Config,
) (exporter.Logs, error) {
	parsedCfg, err := config.NewConfigFrom(&cfg)
	if err != nil {
		return nil, err
	}

	lsOutputCfg := logstashOutputConfig{}
	err = parsedCfg.Unpack(&lsOutputCfg)
	if err != nil {
		return nil, err
	}

	exp := logstashExporter{
		config: &lsOutputCfg,
	}

	return exporterhelper.NewLogs(
		ctx,
		settings,
		cfg,
		exp.pushLogData,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
	)
}

func (l *logstashExporter) pushLogData(context.Context, plog.Logs) error {
	return nil
}
