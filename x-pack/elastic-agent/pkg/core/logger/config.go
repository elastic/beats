// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
)

func configToLogpConfig(cfg *Config) (*logpConfig, error) {
	logsFilePath := filepath.Join(paths.Home(), "data", "logs")
	if err := os.MkdirAll(logsFilePath, 0750); err != nil {
		return nil, err
	}

	logpConfig := defaultLogpConfig()

	// to std by default
	logpConfig.ToStderr = true

	if cfg != nil {
		logpConfig.Level = logp.Level(cfg.Level)

		if cfg.Output == FileOutput {
			logpConfig.Beat = agentName
			logpConfig.Files.Path = logsFilePath
			logpConfig.Files.Name = agentName
			logpConfig.ECSEnabled = true
			logpConfig.ToFiles = true
			logpConfig.ToStderr = false

		}
	}

	return &logpConfig, nil
}

func logpLevel(level loggingLevel) logp.Level {
	logpLevel := logp.DebugLevel

	switch level.String() {
	case "debug":
		logpLevel = logp.DebugLevel
	case "info":
		logpLevel = logp.InfoLevel
	case "error":
		logpLevel = logp.ErrorLevel
	}

	return logpLevel
}

// Config is a configuration of logging.
type Config struct {
	Level  loggingLevel  `config:"level"`
	Output loggingOutput `config:"output"`
}

// DefaultLoggingConfig creates a default logging configuration.
func DefaultLoggingConfig() *Config {
	return &Config{
		Level:  loggingLevel(logp.DebugLevel),
		Output: ConsoleOutput,
	}
}

type loggingLevel logp.Level

var loggingLevelMap = map[string]loggingLevel{
	"trace": loggingLevel(logp.DebugLevel), // For backward compatibility
	"debug": loggingLevel(logp.DebugLevel),
	"info":  loggingLevel(logp.InfoLevel),
	"error": loggingLevel(logp.ErrorLevel),
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

func (m *loggingLevel) MarshalYAML() (interface{}, error) {
	return m.String(), nil
}

func (m *loggingLevel) MarshalJSON() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m *loggingLevel) String() string {
	for s, v := range loggingLevelMap {
		if v == *m {
			return s
		}
	}

	return "unknown"
}

type loggingOutput uint8

const (
	ConsoleOutput loggingOutput = iota
	FileOutput
)

var loggingOutputMap = map[string]loggingOutput{
	"file":    FileOutput,
	"console": ConsoleOutput,
}

func (m *loggingOutput) Unpack(v string) error {
	mgt, ok := loggingOutputMap[v]
	if !ok {
		return fmt.Errorf(
			"unknown logging level mode, received '%s' and valid values are 'trace', 'debug', 'info' or 'error'",
			v,
		)
	}
	*m = mgt
	return nil
}

func (m *loggingOutput) MarshalYAML() (interface{}, error) {
	return m.String(), nil
}

func (m *loggingOutput) MarshalJSON() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m *loggingOutput) String() string {
	for s, v := range loggingOutputMap {
		if v == *m {
			return s
		}
	}

	return "unknown"
}
