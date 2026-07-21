// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatprocessor

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"go.opentelemetry.io/collector/component"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

// allowedProcessors is the list of Beat processor names that may be configured
// in the OTel Beat processor.
var allowedProcessors = []string{
	"add_cloud_metadata",
	"add_docker_metadata",
	"add_fields",
	"add_host_metadata",
	"add_kubernetes_metadata",
	"detect_mime_type",
}

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

	for _, processorNameAndConfig := range cfg.Processors {
		processor, err := createProcessor(processorNameAndConfig, logpLogger)
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
func createProcessor(processorNameAndConfig map[string]any, logpLogger *logp.Logger) (beat.Processor, error) {
	if len(processorNameAndConfig) == 0 {
		return nil, nil
	}
	if len(processorNameAndConfig) > 1 {
		if len(processorNameAndConfig) < 10 {
			configKeys := make([]string, 0, len(processorNameAndConfig))
			for k := range processorNameAndConfig {
				configKeys = append(configKeys, k)
			}
			return nil, fmt.Errorf("expected single processor name but got %v: %v", len(processorNameAndConfig), configKeys)
		}
		return nil, fmt.Errorf("expected single processor name but got %v", len(processorNameAndConfig))
	}

	for processorName, processorConfig := range processorNameAndConfig {
		processorConfig, configError := config.NewConfigFrom(processorConfig)
		if configError != nil {
			return nil, fmt.Errorf("failed to create config for processor '%s': %w", processorName, configError)
		}

		if !slices.Contains(allowedProcessors, processorName) {
			return nil, fmt.Errorf("invalid processor name '%s'", processorName)
		}

		constructor, err := processors.GetConstructor(processorName)
		if err != nil {
			return nil, fmt.Errorf("failed to get constructor for '%s': %w", processorName, err)
		}

		// no need to wrap NewConditional because it is being wrapped when processors are registered
		processorInstance, createProcessorError := constructor(processorConfig, logpLogger)
		if createProcessorError != nil {
			return nil, fmt.Errorf("failed to create processor '%s': %w", processorName, createProcessorError)
		}

		return processorInstance, nil
	}

	return nil, errors.New("malformed processor config")
}

func (p *beatProcessor) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Shutdown closes every processor that was constructed for this component.
func (p *beatProcessor) Shutdown(_ context.Context) error {
	var errs []error
	for _, proc := range p.processors {
		if err := processors.Close(proc); err != nil {
			errs = append(errs, err)
		}
	}
	// Drop the references so a repeated Shutdown does not double-close.
	p.processors = nil
	return errors.Join(errs...)
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

	beatEvent.Fields = otelmap.ToMapstr(logRecord.Body().Map())

	return beatEvent, nil
}

func packBeatEventIntoOTelLogRecord(beatEvent *beat.Event, logRecord plog.LogRecord) error {
	return otelmap.FromMapstr(logRecord.Body().Map(), beatEvent.Fields)
}
