// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstashexporter

import (
	"go.opentelemetry.io/collector/component"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"

	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/logstash"
	"github.com/elastic/elastic-agent-libs/config"
)

type Config map[string]any

type logstashOutputConfig struct {
	outputs.HostWorkerCfg `config:",inline"`
	logstash.Config       `config:",inline"`
}

func createDefaultConfig() component.Config {
	defaultConfig, err := config.NewConfigFrom(logstashOutputConfig{Config: logstash.DefaultConfig()})
	if err != nil {
		return nil
	}
	var configMap map[string]any
	if err = defaultConfig.Unpack(&configMap); err != nil {
		return &Config{}
	}
	return Config(configMap)
}

func parseLogstashConfig(cfg *component.Config) (*config.C, *logstashOutputConfig, error) {
	rawConfig, err := config.NewConfigFrom(&cfg)
	if err != nil {
		return nil, nil, err
	}
	parsedConfig, err := unpackLogstashConfig(rawConfig)
	if err != nil {
		return nil, nil, err
	}
	return rawConfig, parsedConfig, nil
}

func unpackLogstashConfig(cfg *config.C) (*logstashOutputConfig, error) {
	err := cfgwarn.CheckRemoved6xSettings(cfg, "port")
	if err != nil {
		return nil, err
	}
	parsed := logstashOutputConfig{}
	if err = cfg.Unpack(&parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}
