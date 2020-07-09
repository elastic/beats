// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"go.elastic.co/ecszap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/file"
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
	internal, err := makeInternalFileOutput()
	if err != nil {
		return nil, err
	}
	if err := configure.Logging("", cfg, newMultiCoreWrapper(internal)); err != nil {
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
	cfg.Level = logp.DebugLevel
	cfg.Files.Path = paths.Logs()
	cfg.Files.Name = fmt.Sprintf("%s.log", agentName)

	return &cfg
}

// makeInternalFileOutput creates a zapcore.Core logger that cannot be changed with configuration.
//
// This is the logger that the spawned filebeat expects to read the log file from and ship to ES.
func makeInternalFileOutput() (zapcore.Core, error) {
	// defaultCfg is used to set the defaults for the file rotation of the internal logging
	// these settings cannot be changed by a user configuration
	defaultCfg := logp.DefaultConfig(logp.DefaultEnvironment)
	filename := filepath.Join(paths.Data(), "logs", fmt.Sprintf("%s-json.log", agentName))

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
	return ecszap.WrapCore(zapcore.NewCore(encoder, rotator, logp.DebugLevel.ZapLevel())), nil
}

// newMultiCoreWrapper takes the current zapcore.Core and appends it with the provided others.
func newMultiCoreWrapper(cores ...zapcore.Core) func (zapcore.Core) zapcore.Core {
	return func(core zapcore.Core) zapcore.Core {
		return &multiCore{append(cores, core)}
	}
}

// multiCore allows multiple cores to be used for logging.
type multiCore struct {
	cores []zapcore.Core
}

// Enabled returns true if the level is enabled in any one of the cores.
func (m multiCore) Enabled(level zapcore.Level) bool {
	for _, core := range m.cores {
		if core.Enabled(level) {
			return true
		}
	}
	return false
}

// With creates a new multiCore with each core set with the given fields.
func (m multiCore) With(fields []zapcore.Field) zapcore.Core {
	cores := make([]zapcore.Core, len(m.cores))
	for i, core := range m.cores {
		cores[i] = core.With(fields)
	}
	return &multiCore{cores}
}

// Check will place each core that checks for that entry.
func (m multiCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	for _, core := range m.cores {
		checked = core.Check(entry, checked)
	}
	return checked
}

// Write writes the entry to each core.
func (m multiCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	var errs error
	for _, core := range m.cores {
		if err := core.Write(entry, fields); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

// Sync syncs each core.
func (m multiCore) Sync() error {
	var errs error
	for _, core := range m.cores {
		if err := core.Sync(); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}
