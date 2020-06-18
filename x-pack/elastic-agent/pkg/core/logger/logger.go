// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logger

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/logp/configure"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

const agentName = "elastic-agent"

// Logger alias ecslog.Logger with Logger.
type Logger = logp.Logger

// Config is a logging config.
type Config = logp.Config

// New returns a configured ECS Logger
func New(name string) (*Logger, error) {
	dc, err := defaultConfig()
	if err != nil {
		return nil, err
	}
	return new(name, dc)
}

// NewWithLogpLevel returns a configured logp Logger with specified level.
func NewWithLogpLevel(name string, level logp.Level) (*Logger, error) {
	cfg := struct {
		Level string `config:"level"`
	}{Level: level.String()}

	commonCfg, err := common.NewConfigFrom(cfg)
	if err != nil {
		return nil, err
	}

	return new(name, commonCfg)
}

//NewFromConfig takes the user configuration and generate the right logger.
// TODO: Finish implementation, need support on the library that we use.
func NewFromConfig(name string, cfg *config.Config) (*Logger, error) {
	defaultCfg, err := defaultConfig()
	if err != nil {
		return nil, err
	}
	wrappedConfig := &struct {
		Logging *common.Config `config:"logging"`
	}{Logging: defaultCfg}

	if err := cfg.Unpack(&wrappedConfig); err != nil {
		return nil, err
	}

	return new(name, wrappedConfig.Logging)
}

func new(name string, cfg *common.Config) (*Logger, error) {
	if err := configure.Logging("", cfg); err != nil {
		return nil, fmt.Errorf("error initializing logging: %v", err)
	}

	return logp.NewLogger(name), nil
}

func defaultConfig() (*common.Config, error) {
	cfg := DefaultLoggingConfig()

	// work around custom types and common config
	// when custom type is transformed to common.Config
	// value is determined based on reflect value which is incorrect
	// enum vs human readable form
	yamlCfg, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	commonLogp, err := common.NewConfigFrom(string(yamlCfg))
	if err != nil {
		return nil, errors.New(err, errors.TypeConfig)
	}

	return commonLogp, nil
}

// DefaultLoggingConfig returns default configuration for agent logging.
func DefaultLoggingConfig() *Config {
	cfg := logp.DefaultConfig(logp.DefaultEnvironment)
	cfg.Beat = agentName
	cfg.ECSEnabled = true
	cfg.Level = logp.DebugLevel
	cfg.Files.Path = filepath.Join(paths.Home(), "data", "logs")
	cfg.Files.Name = agentName

	return &cfg
}
