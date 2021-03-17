// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"go.elastic.co/ecszap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/logp/configure"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
)

const agentName = "elastic-agent"

// Logger alias ecslog.Logger with Logger.
type Logger = logp.Logger

// Config is a logging config.
type Config = logp.Config

// New returns a configured ECS Logger
func New(name string) (*Logger, error) {
	defaultCfg := DefaultLoggingConfig()
	return new(name, defaultCfg)
}

// NewWithLogpLevel returns a configured logp Logger with specified level.
func NewWithLogpLevel(name string, level logp.Level) (*Logger, error) {
	defaultCfg := DefaultLoggingConfig()
	defaultCfg.Level = level

	return new(name, defaultCfg)
}

// NewFromConfig takes the user configuration and generate the right logger.
// TODO: Finish implementation, need support on the library that we use.
func NewFromConfig(name string, cfg *Config) (*Logger, error) {
	return new(name, cfg)
}

func new(name string, cfg *Config) (*Logger, error) {
	commonCfg, err := toCommonConfig(cfg)
	if err != nil {
		return nil, err
	}

	var outputs []zapcore.Core
	if cfg.ToFiles {
		internal, err := makeInternalFileOutput(cfg)
		if err != nil {
			return nil, err
		}

		outputs = append(outputs, internal)
	}

	if err := configure.LoggingWithOutputs("", commonCfg, outputs...); err != nil {
		return nil, fmt.Errorf("error initializing logging: %v", err)
	}
	return logp.NewLogger(name), nil
}

func toCommonConfig(cfg *Config) (*common.Config, error) {
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
	cfg.Level = logp.InfoLevel
	cfg.ToFiles = true
	cfg.Files.Path = paths.Logs()
	cfg.Files.Name = fmt.Sprintf("%s.log", agentName)

	return &cfg
}

// makeInternalFileOutput creates a zapcore.Core logger that cannot be changed with configuration.
//
// This is the logger that the spawned filebeat expects to read the log file from and ship to ES.
func makeInternalFileOutput(cfg *Config) (zapcore.Core, error) {
	// defaultCfg is used to set the defaults for the file rotation of the internal logging
	// these settings cannot be changed by a user configuration
	defaultCfg := logp.DefaultConfig(logp.DefaultEnvironment)
	filename := filepath.Join(paths.Home(), "logs", fmt.Sprintf("%s-json.log", cfg.Beat))

	rotator, err := file.NewFileRotator(filename,
		file.MaxSizeBytes(defaultCfg.Files.MaxSize),
		file.MaxBackups(defaultCfg.Files.MaxBackups),
		file.Permissions(os.FileMode(defaultCfg.Files.Permissions)),
		file.Interval(defaultCfg.Files.Interval),
		file.RotateOnStartup(defaultCfg.Files.RotateOnStartup),
		file.RedirectStderr(defaultCfg.Files.RedirectStderr),
	)
	if err != nil {
		return nil, errors.New("failed to create internal file rotator")
	}

	encoder := zapcore.NewJSONEncoder(ecszap.ECSCompatibleEncoderConfig(logp.JSONEncoderConfig()))
	return ecszap.WrapCore(zapcore.NewCore(encoder, rotator, cfg.Level.ZapLevel())), nil
}
