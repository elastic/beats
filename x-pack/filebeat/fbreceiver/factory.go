// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"github.com/elastic/beats/v7/filebeat/beater"
	"github.com/elastic/beats/v7/filebeat/cmd"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/x-pack/filebeat/include"
	inputs "github.com/elastic/beats/v7/x-pack/filebeat/input/default-inputs"
	fbOtel "github.com/elastic/beats/v7/x-pack/filebeat/otel"
	xpInstance "github.com/elastic/beats/v7/x-pack/libbeat/cmd/instance"
	"github.com/elastic/elastic-agent-libs/mapstr"

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

func createReceiver(ctx context.Context, set receiver.Settings, baseCfg component.Config, consumer consumer.Logs) (receiver.Logs, error) {
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


	// Initialize the OpenTelemetry tracer provider to enable tracing if configured.
	var tracerProvider trace.TracerProvider
	if tracerProvider, err = fbOtel.TracerProvider(ctx, set.BuildInfo.Version); err != nil {
		set.Logger.Error("failed to initialize OpenTelemetry tracing  %+v", zap.Error(err))
	} else if tracerProvider == nil {
		set.Logger.Info("OpenTelemetry tracing is disabled")
		tracerProvider = noop.TracerProvider{}
	}

	b, err := xpInstance.NewBeatForReceiver(settings, cfg.Beatconfig, consumer, set.ID.String(), set.Logger.Core(), tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("error creating %s: %w", Name, err)
	}

	beatCreator := beater.New(inputs.Init)
	br, err := xpInstance.NewBeatReceiver(ctx, b, beatCreator)
	if err != nil {
		return nil, fmt.Errorf("error creating %s:%w", Name, err)
	}

	return &filebeatReceiver{BeatReceiver: br}, nil
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
