// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstashexporter

import (
	"go.opentelemetry.io/collector/component"

	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/logstash"
	"github.com/elastic/elastic-agent-libs/config"
)

type Config map[string]any

func createDefaultConfig() component.Config {
	return &Config{}
}

type logstashOutputConfig struct {
	outputs.HostWorkerCfg `config:",inline"`
	logstash.Config       `config:",inline"`
}

func parseLogstashConfig(cfg *component.Config) (*config.C, *logstashOutputConfig, error) {
	rawConfig, err := config.NewConfigFrom(&cfg)
	if err != nil {
		return nil, nil, err
	}

	parsedConfig := logstashOutputConfig{}
	err = rawConfig.Unpack(&parsedConfig)
	if err != nil {
		return nil, nil, err
	}

	return rawConfig, &parsedConfig, nil
}
