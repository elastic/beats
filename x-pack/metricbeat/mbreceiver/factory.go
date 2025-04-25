// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/otelbeat/beatreceiver"
	"github.com/elastic/beats/v7/metricbeat/beater"
	"github.com/elastic/beats/v7/metricbeat/cmd"
	"github.com/elastic/elastic-agent-libs/config"
)

const (
	Name = "metricbeatreceiver"
)

func createDefaultConfig() component.Config {
	return &Config{}
}

func createReceiver(_ context.Context, set receiver.Settings, baseCfg component.Config, consumer consumer.Logs) (receiver.Logs, error) {
	cfg, ok := baseCfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("could not convert otel config to metricbeat config")
	}
	settings := cmd.MetricbeatSettings(Name)
	settings.ElasticLicensed = true

	b, err := instance.NewBeatReceiver(settings, cfg.Beatconfig, false, consumer, set.Logger.Core())
	if err != nil {
		return nil, fmt.Errorf("error creating %s: %w", Name, err)
	}

	beatCreator := beater.DefaultCreator()

	beatConfig, err := b.BeatConfig()
	if err != nil {
		return nil, fmt.Errorf("error getting beat config: %w", err)
	}

	mbBeater, err := beatCreator(&b.Beat, beatConfig)
	if err != nil {
		return nil, fmt.Errorf("error getting %s creator:%w", Name, err)
	}

	httpConf := struct {
		HTTP *config.C `config:"http"`
	}{}
	if err := b.RawConfig.Unpack(&httpConf); err != nil {
		return nil, fmt.Errorf("error unpacking monitoring config: %w", err)
	}

	beatReceiver := beatreceiver.BeatReceiver{
		Beat:     b,
		Beater:   mbBeater,
		Logger:   set.Logger,
		HttpConf: httpConf.HTTP,
	}
	return &metricbeatReceiver{BeatReceiver: beatReceiver}, nil
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		component.MustNewType(Name),
		createDefaultConfig,
		receiver.WithLogs(createReceiver, component.StabilityLevelAlpha))
}
