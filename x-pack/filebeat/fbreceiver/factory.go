// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/filebeat/beater"
	"github.com/elastic/beats/v7/filebeat/cmd"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/x-pack/filebeat/include"
	inputs "github.com/elastic/beats/v7/x-pack/filebeat/input/default-inputs"
	xpInstance "github.com/elastic/beats/v7/x-pack/libbeat/cmd/instance"

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
	settings.Processing = processing.MakeDefaultSupport(true, nil, processing.WithECS, processing.WithHost, processing.WithAgentMeta())
	settings.ElasticLicensed = true
	settings.Initialize = append(settings.Initialize, include.InitializeModule)

	b, err := xpInstance.NewBeatForReceiver(settings, cfg.Beatconfig, consumer, set.ID.String(), set.Logger.Core())
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

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		component.MustNewType(Name),
		createDefaultConfig,
		receiver.WithLogs(createReceiver, component.StabilityLevelAlpha))
}
