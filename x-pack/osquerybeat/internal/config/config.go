// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/processors"
)

// Default index name for ad-hoc queries, since the dataset is defined at the stream level, for example:
// streams:
// - id: '123456'
//   data_stream:
// 	dataset: osquery_manager.result
// 	type: logs
//   query: select * from usb_devices

const DefaultStreamIndex = "logs-osquery_manager.result-default"

type StreamConfig struct {
	ID       string        `config:"id"`
	Query    string        `config:"query"`
	Interval time.Duration `config:"interval"`
	Index    string        `config:"index"` // ES output index pattern
}

type InputConfig struct {
	Type       string                  `config:"type"`
	Streams    []StreamConfig          `config:"streams"`
	Processors processors.PluginConfig `config:"processors"`
}

type Config struct {
	Inputs []InputConfig `config:"inputs"`
}

type void struct{}
type inputTypeSet map[string]void

var none = void{}

var DefaultConfig = Config{}

func StreamsFromInputs(inputs []InputConfig) ([]StreamConfig, []string) {
	var (
		streams []StreamConfig
	)

	typeSet := make(inputTypeSet, 1)
	for _, input := range inputs {
		typeSet[input.Type] = none
		for _, s := range input.Streams {
			if s.Index == "" {
				s.Index = DefaultStreamIndex
			}
			streams = append(streams, s)
		}
	}

	var inputTypes []string
	for t := range typeSet {
		inputTypes = append(inputTypes, t)
	}
	return streams, inputTypes
}
