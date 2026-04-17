// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/elastic/beats/v7/heartbeat/beater"
	"github.com/elastic/beats/v7/heartbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"

	// Import OSS monitor types.
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/http"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/icmp"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/tcp"

	// Import X-Pack modules.
	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"

	xpInstance "github.com/elastic/beats/v7/x-pack/libbeat/cmd/instance"
)

const (
	Name = "heartbeatreceiver"
)

type Settings struct {
	Home string
	Data string
}

func createReceiver(ctx context.Context, set receiver.Settings, baseCfg component.Config, consumer consumer.Logs) (receiver.Logs, error) {
	cfg, ok := baseCfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("could not convert otel config to heartbeat config")
	}
	settings := cmd.HeartbeatSettings()
	settings.Processing = processing.MakeDefaultSupport(true, nil, processing.WithECS, processing.WithHost, processing.WithAgentMeta())
	settings.ElasticLicensed = true

	b, err := xpInstance.NewBeatForReceiver(settings, cfg.Beatconfig, consumer, set.ID.String(), set.Logger.Core(), cfg.IncludeMetadata)
	if err != nil {
		return nil, fmt.Errorf("error creating %s: %w", Name, err)
	}

	beatCreator := beater.New
	br, err := xpInstance.NewBeatReceiver(ctx, b, beatCreator, set)
	if err != nil {
		return nil, fmt.Errorf("error creating %s: %w", Name, err)
	}
	return &heartbeatReceiver{BeatReceiver: br}, nil
}

// NewFactory creates a new receiver Factory with empty default paths.
// It is compatible with the OpenTelemetry Collector Builder, which expects
// parameterless NewFactory functions.
func NewFactory() receiver.Factory {
	return NewFactoryWithSettings(Settings{})
}

// NewFactoryWithSettings creates a new receiver Factory.  The supplied
// Settings.Home should be the path that contains the "module"
// directory so modules can be found and loaded.  The supplied
// Settings.Data should point to the directory where state information
// will be kept.  Both can be overridden by passing in path
// information in the configuration when the receiver in instantiated.
// This just provides defaults.
func NewFactoryWithSettings(s Settings) receiver.Factory {
	return receiver.NewFactory(
		component.MustNewType(Name),
		func() component.Config {
			return &Config{
				Beatconfig: map[string]any{
					"path": map[string]any{
						"home": s.Home,
						"data": s.Data,
					},
				},
			}
		},
		receiver.WithLogs(createReceiver, component.StabilityLevelAlpha))
}
