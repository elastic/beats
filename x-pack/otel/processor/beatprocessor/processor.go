// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatprocessor

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otel/eventcache"
	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/actions/addfields"
	"github.com/elastic/beats/v7/libbeat/processors/add_cloud_metadata"
	"github.com/elastic/beats/v7/libbeat/processors/add_docker_metadata"
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

		var constructor processors.Constructor

		switch processorName {
		case "add_cloud_metadata":
			constructor = add_cloud_metadata.New
		case "add_docker_metadata":
			constructor = add_docker_metadata.New
		case "add_fields":
			constructor = addfields.CreateAddFields
		case "add_host_metadata":
			constructor = add_host_metadata.New
		case "add_kubernetes_metadata":
			constructor = add_kubernetes_metadata.New
		default:
			return nil, fmt.Errorf("invalid processor name '%s'", processorName)
		}

		// Wrap the constructor with NewConditional so that `when` conditions
		// configured on the processor are honored.
		processorInstance, createProcessorError := processors.NewConditional(constructor)(processorConfig, logpLogger)
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

func (p *beatProcessor) Shutdown(_ context.Context) error {
	return nil
}

func (p *beatProcessor) ConsumeLogs(_ context.Context, logs plog.Logs) (plog.Logs, error) {
	for _, resourceLogs := range logs.ResourceLogs().All() {
		for _, scopeLogs := range resourceLogs.ScopeLogs().All() {
			for _, logRecord := range scopeLogs.LogRecords().All() {
				if tokenVal, hasToken := logRecord.Attributes().Get(eventcache.TokenAttribute); hasToken {
					// Native-events path: always encode to pcommon regardless of
					// whether beat processors are configured.
					p.processNativeEvent(logRecord, tokenVal.Int())
				} else if len(p.processors) > 0 {
					p.processOTelEvent(logRecord)
				}
			}
		}
	}

	return logs, nil
}

// processNativeEvent handles the native-events fast path: the full beat.Event
// lives in the eventcache. After running beat processors the body is encoded
// from the (possibly modified) event and the cache token attribute is removed.
func (p *beatProcessor) processNativeEvent(logRecord plog.LogRecord, token int64) {
	entry, ok := eventcache.Take(token)
	if !ok {
		p.logger.Error("native event cache miss", zap.Int64("token", token))
		return
	}

	// No clone needed: the publisher pipeline guarantees private map ownership
	// per event when NativeEvents is enabled (addFields clones the shared builtin
	// fields at injection time). All beat processors produce fresh maps per call.
	beatEvent := &beat.Event{
		Timestamp: entry.Event.Content.Timestamp,
		Fields:    entry.Event.Content.Fields,
		Meta:      entry.Event.Content.Meta,
	}

	for _, processor := range p.processors {
		processed, err := processor.Run(beatEvent)
		if err != nil {
			p.logger.Error("error processing Beat event", zap.Error(err))
			continue
		}
		beatEvent = processed
	}

	info := *entry.BeatInfo
	if err := otelmap.EncodeEventBody(logRecord, beatEvent.Fields, beatEvent.Timestamp, beatEvent.Meta, info.Beat, info.Version, info.IncludeMetadata); err != nil {
		p.logger.Error("error encoding native Beat event into OTel log record", zap.Error(err))
	}
	logRecord.Attributes().Remove(eventcache.TokenAttribute)
}

// processOTelEvent is the standard path: the logRecord body already contains
// the full pcommon map and beat processors mutate it in place.
func (p *beatProcessor) processOTelEvent(logRecord plog.LogRecord) {
	beatEvent, err := unpackBeatEventFromOTelLogRecord(logRecord)
	if err != nil {
		p.logger.Error("error converting OTel log to Beat event", zap.Error(err))
		return
	}

	for _, processor := range p.processors {
		processed, err := processor.Run(beatEvent)
		if err != nil {
			p.logger.Error("error processing Beat event", zap.Error(err))
			continue
		}
		beatEvent = processed
	}

	if err := packBeatEventIntoOTelLogRecord(beatEvent, logRecord); err != nil {
		p.logger.Error("error converting processed Beat event to OTel log record", zap.Error(err))
	}
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
