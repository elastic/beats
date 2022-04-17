// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import (
	"github.com/menderesk/beats/v7/libbeat/processors"
)

// Default index name for ad-hoc queries, since the dataset is defined at the stream level, for example:
// streams:
// - id: '123456'
//   data_stream:
// 	dataset: osquery_manager.result
// 	type: logs
//   query: select * from usb_devices

const DefaultNamespace = "default"

const datastreamPrefix = "logs-osquery_manager.result-"

type StreamConfig struct {
	ID         string                 `config:"id"`
	Query      string                 `config:"query"`       // the SQL query to run
	Interval   int                    `config:"interval"`    // an interval in seconds to run the query (subject to splay/smoothing). It has a maximum value of 604,800 (1 week).
	Platform   string                 `config:"platform"`    // restrict this query to a given platform, default is 'all' platforms; you may use commas to set multiple platforms
	Version    string                 `config:"version"`     // only run on osquery versions greater than or equal-to this version string
	ECSMapping map[string]interface{} `config:"ecs_mapping"` // ECS mapping definition where the key is the source field in osquery result and the value is the destination fields in ECS
}

type DatastreamConfig struct {
	Namespace string `config:"namespace"`
}

type InputConfig struct {
	Name       string                  `config:"name"`
	Type       string                  `config:"type"`
	Datastream DatastreamConfig        `config:"data_stream"` // Datastream configuration
	Processors processors.PluginConfig `config:"processors"`

	// Full Osquery configuration
	Osquery *OsqueryConfig `config:"osquery"`

	// Deprecated
	Streams   []StreamConfig `config:"streams"`
	Platform  string         `config:"iplatform"` // restrict all queries to a given platform, default is 'all' platforms; you may use commas to set multiple platforms
	Version   string         `config:"iversion"`  // only run the queries with osquery versions greater than or equal-to this version string
	Discovery []string       `config:"discovery"` // a list of discovery queries https://osquery.readthedocs.io/en/stable/deployment/configuration/#discovery-queries
}

type Config struct {
	Inputs []InputConfig `config:"inputs"`
}

var DefaultConfig = Config{}

func Datastream(namespace string) string {
	if namespace == "" {
		namespace = DefaultNamespace
	}
	return datastreamPrefix + namespace
}

// GetOsqueryOptions Returns options from the first input if available
func GetOsqueryOptions(inputs []InputConfig) map[string]interface{} {
	if len(inputs) == 0 {
		return nil
	}
	if inputs[0].Osquery == nil {
		return nil
	}
	return inputs[0].Osquery.Options
}
