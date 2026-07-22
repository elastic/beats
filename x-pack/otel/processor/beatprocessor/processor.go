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
	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

<<<<<<< HEAD
=======
// allowedProcessors is the list of Beat processor names that may be configured
// in the OTel Beat processor.
var allowedProcessors = []string{
	"add_cloud_metadata",
	"add_docker_metadata",
	"add_fields",
	"add_host_metadata",
	"add_kubernetes_metadata",
	"detect_mime_type",
	"drop_fields",
}

>>>>>>> 5188361d1 (feat: add `drop_fields` processor to OTel Beat processor (#52221))
type beatProcessor struct {
	logger     *zap.Logger
	processors []beat.Processor
	// pdataProcs is non-nil only when every processor in the chain implements
	// processors.PdataProcessor. When set, ConsumeLogs takes the zero-copy
	// pdata fast path; otherwise it falls back to a single legacy round-trip.
	pdataProcs []processors.PdataProcessor
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

	bp.pdataProcs = buildPdataProcs(bp.processors)
	if bp.pdataProcs == nil && len(bp.processors) > 0 {
		var legacy []string
		for _, p := range bp.processors {
			if _, ok := p.(processors.PdataProcessor); !ok {
				legacy = append(legacy, p.String())
			}
		}
		bp.logger.Warn("pdata fast path disabled: processor(s) lack RunPdata, falling back to legacy round-trip per event",
			zap.Strings("legacy_processors", legacy))
	}

	return bp, nil
}

// buildPdataProcs returns a typed slice of PdataProcessor when every proc in
// the list implements the interface, or nil if any one does not.
func buildPdataProcs(procs []beat.Processor) []processors.PdataProcessor {
	pdataProcs := make([]processors.PdataProcessor, 0, len(procs))
	for _, p := range procs {
		pp, ok := p.(processors.PdataProcessor)
		if !ok {
			return nil
		}
		pdataProcs = append(pdataProcs, pp)
	}
	return pdataProcs
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

		switch processorName {
		case "add_host_metadata", "add_cloud_metadata", "add_docker_metadata", "add_kubernetes_metadata", "add_fields":
		default:
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
			scopeLogs.LogRecords().RemoveIf(func(logRecord plog.LogRecord) bool {
				var (
					drop bool
					err  error
				)
				if p.pdataProcs != nil {
					drop, err = p.consumeLogRecordPdata(logRecord)
				} else {
					drop, err = p.consumeLogRecordLegacy(logRecord)
				}
				if err != nil {
					p.logger.Error("error processing Beat event", zap.Error(err))
					return false
				}
				return drop
			})
		}
	}

	return logs, nil
}

// consumeLogRecordPdata runs all processors directly on the log record's
// pcommon.Map body, with no round-trip to mapstr. It is only called when
// every processor in the chain implements processors.PdataProcessor.
func (p *beatProcessor) consumeLogRecordPdata(logRecord plog.LogRecord) (bool, error) {
	body := logRecord.Body().Map()
	for _, proc := range p.pdataProcs {
		drop, err := proc.RunPdata(body)
		if err != nil {
			return false, err
		}
		if drop {
			return true, nil
		}
	}
	return false, nil
}

// consumeLogRecordLegacy unpacks the log record into a beat.Event, runs every
// processor via Run, then packs the result back. It is used when at least one
// processor in the chain does not implement processors.PdataProcessor, so the
// entire chain pays a single round-trip rather than a per-processor one.
func (p *beatProcessor) consumeLogRecordLegacy(logRecord plog.LogRecord) (bool, error) {
	event, err := unpackBeatEventFromOTelLogRecord(logRecord)
	if err != nil {
		return false, err
	}
	for _, proc := range p.processors {
		out, err := proc.Run(event)
		if err != nil {
			return false, err
		}
		if out == nil {
			return true, nil
		}
		event = out
	}
	return false, packBeatEventIntoOTelLogRecord(event, logRecord)
}

func unpackBeatEventFromOTelLogRecord(logRecord plog.LogRecord) (*beat.Event, error) {
	beatEvent := &beat.Event{}
	beatEvent.Timestamp = logRecord.Timestamp().AsTime()

	beatEvent.Meta = mapstr.M{}
	beatEvent.Fields = otelmap.ToMapstr(logRecord.Body().Map())

	// otelconsumer serializes beat.Event.Meta into the pdata body under the
	// "@metadata" key. Extract it into event.Meta so that processors using
	// the @metadata target (e.g. add_fields with target:"@metadata") see and
	// can modify the correct field.
	if raw, err := beatEvent.Fields.GetValue("@metadata"); err == nil {
		switch m := raw.(type) {
		case mapstr.M:
			beatEvent.Meta = m
		case map[string]any:
			beatEvent.Meta = mapstr.M(m)
		}
		_ = beatEvent.Fields.Delete("@metadata")
	}

	return beatEvent, nil
}

func packBeatEventIntoOTelLogRecord(beatEvent *beat.Event, logRecord plog.LogRecord) error {
	// Write Meta back under "@metadata" so it survives the round-trip.
	if len(beatEvent.Meta) > 0 {
		if beatEvent.Fields == nil {
			beatEvent.Fields = mapstr.M{}
		}
		beatEvent.Fields["@metadata"] = beatEvent.Meta
	}

	logRecord.Body().Map().Clear()
	return otelmap.FromMapstr(logRecord.Body().Map(), beatEvent.Fields)
}
