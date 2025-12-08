// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatprocessor

import (
	"context"
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otelbeat/otelmap"
	"github.com/elastic/beats/v7/libbeat/processors/add_host_metadata"
	"github.com/elastic/beats/v7/libbeat/processors/add_kubernetes_metadata"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

type beatProcessor struct {
	logger     *zap.Logger
	processors []beat.Processor
}

func newBeatProcessor(set processor.Settings, cfg *Config) (*beatProcessor, error) {
	bp := &beatProcessor{
		logger:     set.Logger,
		processors: []beat.Processor{},
	}

	logpLogger, err := logp.ConfigureWithCoreLocal(logp.Config{}, set.Logger.Core())
	if err != nil {
		return nil, fmt.Errorf("failed to configure logp logger: %w", err)
	}

	for _, processorConfig := range cfg.Processors {
		processor, err := createProcessor(processorConfig, logpLogger)
		if err != nil {
			return nil, fmt.Errorf("failed to create processor: %w", err)
		}
		if processor != nil {
			bp.processors = append(bp.processors, processor)
			bp.logger.Info("Configured Beat processor", zap.String("processor_name", processor.String()))
		}
	}

	return bp, nil
}

// createProcessor creates a Beat processor using the provided configuration.
// The configuration is expected to be a map with a single key containing the processor name
// and the processor's configuration as the value for that key.
// For example: {"add_host_metadata":{"netinfo":{"enabled":false}}}
func createProcessor(cfg map[string]any, logpLogger *logp.Logger) (beat.Processor, error) {
	if len(cfg) == 0 {
		return nil, nil
	}
	if len(cfg) > 1 {
		if len(cfg) < 10 {
			configKeys := make([]string, 0, len(cfg))
			for k := range cfg {
				configKeys = append(configKeys, k)
			}
			return nil, fmt.Errorf("expected single processor name but got %v: %v", len(cfg), configKeys)
		}
		return nil, fmt.Errorf("expected single processor name but got %v", len(cfg))
	}

	for processorName, processorConfig := range cfg {
		switch processorName {
		case "add_host_metadata":
			return createAddHostMetadataProcessor(processorConfig, logpLogger)
		case "add_kubernetes_metadata":
			return createAddKubernetesMetadataProcessor(processorConfig, logpLogger)
		default:
			return nil, fmt.Errorf("invalid processor name '%s'", processorName)
		}
	}
	return nil, errors.New("malformed processor config")
}

func createAddHostMetadataProcessor(cfg any, logpLogger *logp.Logger) (beat.Processor, error) {
	addHostMetadataConfig, err := config.NewConfigFrom(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create add_host_metadata processor config: %w", err)
	}
	addHostMetadataProcessor, err := add_host_metadata.New(addHostMetadataConfig, logpLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create add_host_metadata processor: %w", err)
	}
	return addHostMetadataProcessor, nil
}

func createAddKubernetesMetadataProcessor(cfg any, logpLogger *logp.Logger) (beat.Processor, error) {
	addKubernetesMetadataConfig, err := config.NewConfigFrom(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create add_kubernetes_metadata processor config: %w", err)
	}
	addKubernetesMetadataProcessor, err := add_kubernetes_metadata.New(addKubernetesMetadataConfig, logpLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create add_kubernetes_metadata processor: %w", err)
	}
	return addKubernetesMetadataProcessor, nil
}

func (p *beatProcessor) ConsumeLogs(_ context.Context, logs plog.Logs) (plog.Logs, error) {
	if len(p.processors) == 0 {
		return logs, nil
	}

	for _, resourceLogs := range logs.ResourceLogs().All() {
		for _, scopeLogs := range resourceLogs.ScopeLogs().All() {
			for _, logRecord := range scopeLogs.LogRecords().All() {
				beatEvent, err := unpackBeatEventFromOTelLogRecord(logRecord)
				if err != nil {
					p.logger.Error("error converting OTel log to Beat event", zap.Error(err))
					continue
				}

				for _, processor := range p.processors {
					processedEvent, err := processor.Run(beatEvent)
					if err != nil {
						p.logger.Error("error processing Beat event", zap.Error(err))
						continue
					}
					beatEvent = processedEvent
				}

				packingError := packBeatEventIntoOTelLogRecord(beatEvent, logRecord)
				if packingError != nil {
					p.logger.Error("error converting processed Beat event to OTel log record", zap.Error(packingError))
				}
			}
		}
	}

	return logs, nil
}

func unpackBeatEventFromOTelLogRecord(logRecord plog.LogRecord) (*beat.Event, error) {
	beatEvent := &beat.Event{}
	beatEvent.Timestamp = logRecord.Timestamp().AsTime()

	beatEvent.Meta = mapstr.M{}

	beatEvent.Fields = logRecord.Body().Map().AsRaw()

	return beatEvent, nil
}

func packBeatEventIntoOTelLogRecord(beatEvent *beat.Event, logRecord plog.LogRecord) error {
	beatEvent.Fields = beatEvent.Fields.Clone()
	otelmap.ConvertNonPrimitive((map[string]any)(beatEvent.Fields))
	err := logRecord.Body().Map().FromRaw(beatEvent.Fields)
	return err
}
