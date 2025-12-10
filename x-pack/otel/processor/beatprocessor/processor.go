// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatprocessor

import (
	"context"
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors/add_host_metadata"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"go.opentelemetry.io/collector/pdata/pcommon"
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

<<<<<<< HEAD
	for _, processorConfig := range cfg.Processors {
		processor, err := createProcessor(processorConfig)
=======
	logpLogger, err := logp.ConfigureWithCoreLocal(logp.Config{}, set.Logger.Core())
	if err != nil {
		return nil, fmt.Errorf("failed to configure logp logger: %w", err)
	}

	for _, processorNameAndConfig := range cfg.Processors {
		processor, err := createProcessor(processorNameAndConfig, logpLogger)
>>>>>>> 8458c5a1d (refactor: remove code duplication (#48013))
		if err != nil {
			return nil, fmt.Errorf("failed to create processor: %w", err)
		}
		if processor != nil {
			bp.processors = append(bp.processors, processor)
		}
	}

	return bp, nil
}

<<<<<<< HEAD
func createProcessor(cfg map[string]any) (beat.Processor, error) {
	if len(cfg) == 0 {
=======
// createProcessor creates a Beat processor using the provided configuration.
// The configuration is expected to be a map with a single key containing the processor name
// and the processor's configuration as the value for that key.
// For example: {"add_host_metadata":{"netinfo":{"enabled":false}}}
func createProcessor(processorNameAndConfig map[string]any, logpLogger *logp.Logger) (beat.Processor, error) {
	if len(processorNameAndConfig) == 0 {
>>>>>>> 8458c5a1d (refactor: remove code duplication (#48013))
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
<<<<<<< HEAD
	for processorName, processorConfig := range cfg {
		switch processorName {
		case "add_host_metadata":
			return createAddHostMetadataProcessor(processorConfig)
=======

	for processorName, processorConfig := range processorNameAndConfig {
		processorConfig, configError := config.NewConfigFrom(processorConfig)
		if configError != nil {
			return nil, fmt.Errorf("failed to create config for processor '%s': %w", processorName, configError)
		}

		var processorInstance beat.Processor
		var createProcessorError error

		switch processorName {
		case "add_host_metadata":
			processorInstance, createProcessorError = add_host_metadata.New(processorConfig, logpLogger)
		case "add_kubernetes_metadata":
			processorInstance, createProcessorError = add_kubernetes_metadata.New(processorConfig, logpLogger)
>>>>>>> 8458c5a1d (refactor: remove code duplication (#48013))
		default:
			return nil, fmt.Errorf("invalid processor name '%s'", processorName)
		}

		if createProcessorError != nil {
			return nil, fmt.Errorf("failed to create processor '%s': %w", processorName, createProcessorError)
		}

		return processorInstance, nil
	}

	return nil, errors.New("malformed processor config")
}

<<<<<<< HEAD
func createAddHostMetadataProcessor(cfg any) (beat.Processor, error) {
	addHostMetadataConfig, err := config.NewConfigFrom(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create add_host_metadata processor config: %w", err)
	}
	addHostMetadataProcessor, err := add_host_metadata.New(addHostMetadataConfig, logp.NewLogger("beatprocessor"))
	if err != nil {
		return nil, fmt.Errorf("failed to create add_host_metadata processor: %w", err)
	}
	return addHostMetadataProcessor, nil
}

=======
>>>>>>> 8458c5a1d (refactor: remove code duplication (#48013))
func (p *beatProcessor) ConsumeLogs(_ context.Context, logs plog.Logs) (plog.Logs, error) {
	if len(p.processors) == 0 {
		return logs, nil
	}

	for _, hostProcessor := range p.processors {
		dummyEvent := &beat.Event{}
		dummyEvent.Fields = mapstr.M{}
		dummyEvent.Meta = mapstr.M{}
		dummyEventWithHostMetadata, err := hostProcessor.Run(dummyEvent)
		if err != nil {
			p.logger.Error("error processing host metadata", zap.Error(err))
			continue
		}
		hostMap, ok := dummyEventWithHostMetadata.Fields["host"].(mapstr.M)
		if !ok {
			p.logger.Error("error casting host metadata to mapstr.M", zap.Error(err))
			continue
		}
		otelMap, err := toOtelMap(&hostMap)
		if err != nil {
			p.logger.Error("error converting host metadata", zap.Error(err))
			continue
		}
		for _, resourceLogs := range logs.ResourceLogs().All() {
			for _, scopeLogs := range resourceLogs.ScopeLogs().All() {
				for _, logRecord := range scopeLogs.LogRecords().All() {
					bodyMap := logRecord.Body().Map().PutEmptyMap("host")
					otelMap.CopyTo(bodyMap)
				}
			}
		}
	}

	return logs, nil
}

func toOtelMap(m *mapstr.M) (pcommon.Map, error) {
	otelMap := pcommon.NewMap()
	for key, value := range *m {
		switch typedValue := value.(type) {
		case mapstr.M:
			subMap, err := toOtelMap(&typedValue)
			if err != nil {
				return pcommon.Map{}, fmt.Errorf("failed to convert map for key '%s': %w", key, err)
			}
			otelSubMap := otelMap.PutEmptyMap(key)
			subMap.MoveTo(otelSubMap)
		case []string:
			otelValue := otelMap.PutEmptySlice(key)
			for _, item := range typedValue {
				otelValue.AppendEmpty().SetStr(item)
			}
		default:
			otelValue := otelMap.PutEmpty(key)
			err := otelValue.FromRaw(typedValue)
			if err != nil {
				return pcommon.Map{}, fmt.Errorf("failed to convert value for key '%s': %w", key, err)
			}
		}
	}
	return otelMap, nil
}
