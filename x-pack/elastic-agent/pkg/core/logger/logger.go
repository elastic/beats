// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logger

import (
	"fmt"

	"github.com/urso/ecslog"
	"github.com/urso/ecslog/backend"
	"github.com/urso/ecslog/backend/appender"
	"github.com/urso/ecslog/backend/layout"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

// Logger alias ecslog.Logger with Logger.
type Logger = ecslog.Logger

// Config is a configuration of logging.
type Config struct {
	Level loggingLevel `config:"level"`
}

// DefaultLoggingConfig creates a default logging configuration.
func DefaultLoggingConfig() *Config {
	return &Config{
		Level: loggingLevel(backend.Trace),
	}
}

// New returns a configured ECS Logger
func New() (*Logger, error) {
	return new(backend.Trace)
}

func createJSONBackend(lvl backend.Level) (backend.Backend, error) {
	return appender.Console(lvl, layout.Text(true))
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

	return new(backend.Level(wrappedConfig.Logging.Level))
}

func new(lvl backend.Level) (*Logger, error) {
	backend, err := createJSONBackend(lvl)
	if err != nil {
		return nil, err
	}
	return ecslog.New(backend), nil
}

type loggingLevel backend.Level

var loggingLevelMap = map[string]loggingLevel{
	"trace": loggingLevel(backend.Trace),
	"debug": loggingLevel(backend.Debug),
	"info":  loggingLevel(backend.Info),
	"error": loggingLevel(backend.Error),
}

func (m *loggingLevel) Unpack(v string) error {
	mgt, ok := loggingLevelMap[v]
	if !ok {
		return fmt.Errorf(
			"unknown logging level mode, received '%s' and valid values are 'trace', 'debug', 'info' or 'error'",
			v,
		)
	}
	*m = mgt
	return nil
}

func (m *loggingLevel) String() string {
	for s, v := range loggingLevelMap {
		if v == *m {
			return s
		}
	}

	return "unknown"
}
