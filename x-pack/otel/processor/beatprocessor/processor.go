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

	for _, processorConfig := range cfg.Processors {
		processor, err := createProcessor(processorConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create processor: %w", err)
		}
		if processor != nil {
			bp.processors = append(bp.processors, processor)
		}
	}

	return bp, nil
}

func createProcessor(cfg map[string]any) (beat.Processor, error) {
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
			return createAddHostMetadataProcessor(processorConfig)
		default:
			return nil, fmt.Errorf("invalid processor name '%s'", processorName)
		}
	}
	return nil, errors.New("malformed processor config")
}

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
