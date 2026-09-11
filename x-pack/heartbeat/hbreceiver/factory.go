// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"context"
	"fmt"

	conf "github.com/elastic/elastic-agent-libs/config"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/elastic/beats/v7/heartbeat/beater"
	"github.com/elastic/beats/v7/heartbeat/cmd"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/libbeat/beat"

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

type heartbeatCreator struct {
	creator   beat.Creator
	heartbeat *beater.Heartbeat
}

func (c *heartbeatCreator) create(b *beat.Beat, cfg *conf.C) (beat.Beater, error) {
	created, err := c.creator(b, cfg)
	if err != nil {
		return nil, err
	}

	heartbeat, ok := created.(*beater.Heartbeat)
	if !ok || heartbeat == nil {
		return nil, fmt.Errorf("heartbeat creator returned %T, expected *beater.Heartbeat", created)
	}
	c.heartbeat = heartbeat
	return created, nil
}

func elasticsearchClientStartHook(reference string, heartbeat *beater.Heartbeat) func(component.Host) error {
	return func(host component.Host) error {
		if reference == "" {
			return nil
		}

		var extensionID component.ID
		if err := extensionID.UnmarshalText([]byte(reference)); err != nil {
			return fmt.Errorf("invalid elasticsearch_client component ID %q: %w", reference, err)
		}

		extension, ok := host.GetExtensions()[extensionID]
		if !ok {
			return fmt.Errorf("elasticsearch_client extension %q not found", extensionID.String())
		}
		requester, ok := extension.(monitorstate.ElasticsearchRequester)
		if !ok {
			return fmt.Errorf(
				"elasticsearch_client extension %q has type %T, which does not implement monitorstate.ElasticsearchRequester",
				extensionID.String(),
				extension,
			)
		}

		if heartbeat == nil {
			return fmt.Errorf("heartbeat instance was not captured for elasticsearch_client extension %q", extensionID.String())
		}

		heartbeat.WithElasticsearchStateLoader(requester)
		return nil
	}
}

func createReceiver(ctx context.Context, set receiver.Settings, baseCfg component.Config, consumer consumer.Logs) (receiver.Logs, error) {
	cfg, ok := baseCfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("could not convert otel config to heartbeat config")
	}
	settings := cmd.HeartbeatSettings()
	settings.ElasticLicensed = true

	b, err := xpInstance.NewBeatForReceiver(settings, cfg.Beatconfig, consumer, set.ID.String(), set.Logger.Core())
	if err != nil {
		return nil, fmt.Errorf("error creating %s: %w", Name, err)
	}

	baseCreator := beater.New
	creator := &heartbeatCreator{creator: baseCreator}
	br, err := xpInstance.NewBeatReceiver(ctx, b, creator.create, set)
	if err != nil {
		return nil, fmt.Errorf("error creating %s: %w", Name, err)
	}
	br.SetStartHook(elasticsearchClientStartHook(cfg.ElasticsearchClient, creator.heartbeat))
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
