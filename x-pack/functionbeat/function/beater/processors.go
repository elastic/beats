// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common/fmtstr"
	"github.com/elastic/beats/v8/libbeat/processors"
	"github.com/elastic/beats/v8/libbeat/processors/add_formatted_index"
)

func processorsForFunction(beatInfo beat.Info, config fnExtraConfig) (*processors.Processors, error) {
	procs := processors.NewList(nil)

	// Processor ordering is important:
	// 1. Index configuration
	if !config.Index.IsEmpty() {
		staticFields := fmtstr.FieldsForBeat(beatInfo.Beat, beatInfo.Version)
		timestampFormat, err :=
			fmtstr.NewTimestampFormatString(&config.Index, staticFields)
		if err != nil {
			return nil, err
		}
		indexProcessor := add_formatted_index.New(timestampFormat)
		procs.AddProcessor(indexProcessor)
	}

	// 2. User processors
	userProcessors, err := processors.New(config.Processors)
	if err != nil {
		return nil, err
	}
	procs.AddProcessors(*userProcessors)

	return procs, nil
}
