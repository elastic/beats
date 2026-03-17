// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatprocessor

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"

	"github.com/elastic/beats/v7/x-pack/otel/internal/sharedcomponent"
)

const (
	Name = "beat"
)

// sharedProcessors is the map of shared beatProcessor instances, keyed by *Config
// pointer. When the same component ID is referenced in multiple pipelines the
// OTel framework passes the same *Config pointer to every createLogsProcessor
// call, so pointer equality is the correct identity here.
//
// This avoids spinning up duplicate expensive Beat sub-sharedProcessors (e.g.
// add_cloud_metadata, add_kubernetes_metadata) for every pipeline.
var sharedProcessors = sharedcomponent.NewMap[*Config, *beatProcessor]()

func NewFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType(Name),
		createDefaultConfig,
		processor.WithLogs(createLogsProcessor, component.StabilityLevelDevelopment),
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	beatProcessorConfig, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("failed to cast component config to Beat processor config")
	}

	// LoadOrStore creates the beatProcessor only on the first call for a given
	// *Config. Subsequent calls for the same config (i.e. a second pipeline)
	// return the already-created instance.
	shared, err := sharedProcessors.LoadOrStore(beatProcessorConfig, func() (*beatProcessor, error) {
		return newBeatProcessor(set, beatProcessorConfig)
	})
	if err != nil {
		return nil, err
	}

	// Each pipeline gets its own processorhelper wrapper (with its own
	// nextConsumer), but all wrappers call the same shared ConsumeLogs.
	// Start/Shutdown are delegated to the shared component so the underlying
	// beatProcessor is started and stopped exactly once.
	return processorhelper.NewLogs(
		ctx,
		set,
		cfg,
		nextConsumer,
		shared.Unwrap().ConsumeLogs,
		processorhelper.WithStart(shared.Start),
		processorhelper.WithShutdown(shared.Shutdown),
	)
}
