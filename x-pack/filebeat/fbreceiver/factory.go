// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/filebeat/beater"
	"github.com/elastic/beats/v7/filebeat/cmd"
	"github.com/elastic/beats/v7/libbeat/api"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/otelbeat/beatreceiver"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/beats/v7/x-pack/filebeat/include"
	inputs "github.com/elastic/beats/v7/x-pack/filebeat/input/default-inputs"
	"github.com/elastic/elastic-agent-libs/mapstr"
	metricreport "github.com/elastic/elastic-agent-system-metrics/report"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

const (
	Name = "filebeatreceiver"
)

func createDefaultConfig() component.Config {
	return &Config{}
}

func createReceiver(_ context.Context, set receiver.Settings, baseCfg component.Config, consumer consumer.Logs) (receiver.Logs, error) {
	cfg, ok := baseCfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("could not convert otel config to filebeat config")
	}

	settings := cmd.FilebeatSettings(Name)
	globalProcs, err := processors.NewPluginConfigFromList(defaultProcessors())
	if err != nil {
		return nil, fmt.Errorf("error making global processors: %w", err)
	}
	settings.Processing = processing.MakeDefaultSupport(true, globalProcs, processing.WithECS, processing.WithHost, processing.WithAgentMeta())
	settings.ElasticLicensed = true
	settings.Initialize = append(settings.Initialize, include.InitializeModule)

	b, err := instance.NewBeatReceiver(settings, cfg.Beatconfig, true, consumer, set.Logger.Core())
	if err != nil {
		return nil, fmt.Errorf("error creating %s: %w", Name, err)
	}

	beatCreator := beater.New(inputs.Init)

	beatConfig, err := b.BeatConfig()
	if err != nil {
		return nil, fmt.Errorf("error getting beat config: %w", err)
	}

	b.RegisterMetrics()

	statsReg := b.Info.Monitoring.StatsRegistry

	// stats.beat
	processReg := statsReg.GetRegistry("beat")
	if processReg == nil {
		processReg = statsReg.NewRegistry("beat")
	}

	// stats.system
	systemReg := statsReg.GetRegistry("system")
	if systemReg == nil {
		systemReg = statsReg.NewRegistry("system")
	}

	err = metricreport.SetupMetrics(b.Info.Logger.Named("metrics"), b.Info.Beat, version.GetDefaultVersion(), metricreport.WithProcessRegistry(processReg), metricreport.WithSystemRegistry(systemReg))
	if err != nil {
		return nil, fmt.Errorf("error setting up metrics report: %w", err)
	}

	if b.Config.HTTP.Enabled() {
		var err error
		b.API, err = api.NewWithDefaultRoutes(b.Info.Logger.Named("metrics.http"), b.Config.HTTP, api.NamespaceLookupFunc())
		if err != nil {
			return nil, fmt.Errorf("could not start the HTTP server for the API: %w", err)
		}
		b.API.Start()
	}

	fbBeater, err := beatCreator(&b.Beat, beatConfig)
	if err != nil {
		return nil, fmt.Errorf("error getting %s creator:%w", Name, err)
	}

	base := beatreceiver.BeatReceiver{
		Beat:   b,
		Beater: fbBeater,
		Logger: set.Logger,
	}

	return &filebeatReceiver{BeatReceiver: base}, nil
}

// copied from filebeat cmd.
func defaultProcessors() []mapstr.M {
	// processors:
	// - add_host_metadata:
	// 	when.not.contains.tags: forwarded
	// - add_cloud_metadata: ~
	// - add_docker_metadata: ~
	// - add_kubernetes_metadata: ~

	return []mapstr.M{
		{
			"add_host_metadata": mapstr.M{
				"when.not.contains.tags": "forwarded",
			},
		},
		{"add_cloud_metadata": nil},
		{"add_docker_metadata": nil},
		{"add_kubernetes_metadata": nil},
	}
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		component.MustNewType(Name),
		createDefaultConfig,
		receiver.WithLogs(createReceiver, component.StabilityLevelAlpha))
}
