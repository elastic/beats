// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logger

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/logp/configure"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"gopkg.in/yaml.v2"
)

const agentName = "elastic-agent"

// Logger alias ecslog.Logger with Logger.
type Logger = logp.Logger

// New returns a configured ECS Logger
func New() (*Logger, error) {
	return new(DefaultLoggingConfig())
}

// NewWithLogpLevel returns a configured logp Logger with specified level.
func NewWithLogpLevel(level logp.Level) (*Logger, error) {
	dc := DefaultLoggingConfig()
	dc.Level = loggingLevel(level)

	return new(dc)
}

//NewFromConfig takes the user configuration and generate the right logger.
// TODO: Finish implementation, need support on the library that we use.
func NewFromConfig(cfg *config.Config) (*Logger, error) {
	wrappedConfig := &struct {
		Logging *Config `config:"logging"`
	}{
		Logging: DefaultLoggingConfig(),
	}

	if err := cfg.Unpack(&wrappedConfig); err != nil {
		return nil, err
	}

	return new(wrappedConfig.Logging)
}

func new(cfg *Config) (*Logger, error) {
	logpCfg, err := configToLogpConfig(cfg)
	if err != nil {
		return nil, err
	}

	// work around custom types and common config
	yamlCfg, err := yaml.Marshal(logpCfg)
	if err != nil {
		return nil, err
	}

	commonLogp, err := common.NewConfigFrom(string(yamlCfg))
	if err != nil {
		return nil, errors.New(err, errors.TypeConfig)
	}

	if err := configure.Logging("", commonLogp); err != nil {
		return nil, fmt.Errorf("error initializing logging: %v", err)
	}

	return logp.NewLogger(""), nil
}
