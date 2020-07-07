// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logger

import (
	"fmt"
	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/hashicorp/go-multierror"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
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
	cfg.ToStderr = true

	return &cfg
}

func makeInternalFileOutput(cfg *Config) (zapcore.Core, error) {
	// defaultCfg is used to set the defaults for the file rotation of the internal logging
	// these settings cannot be changed by a user configuration
	defaultCfg := logp.DefaultConfig(logp.DefaultEnvironment)
	filename := filepath.Join(paths.Home(), "data", "logs", agentName))

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

func newMultiCoreWrapper(cores ...zapcore.Core) func (zapcore.Core) zapcore.Core {
	return func(core zapcore.Core) zapcore.Core {
		return &multiCore{append(cores, core)}
	}
}

type multiCore struct {
	cores []zapcore.Core
}

func (m multiCore) Enabled(level zapcore.Level) bool {
	for _, core := range m.cores {
		if core.Enabled(level) {
			return true
		}
	}
	return false
}

func (m multiCore) With(fields []zapcore.Field) zapcore.Core {
	
	panic("implement me")
}

func (m multiCore) Check(entry zapcore.Entry, entry2 *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	panic("implement me")
}

func (m multiCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	var errs error
	for _, core := range m.cores {
		if err := core.Write(entry, fields); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

func (m multiCore) Sync() error {
	var errs error
	for _, core := range m.cores {
		if err := core.Sync(); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

// With converts error fields into ECS compliant errors
// before adding them to the logger.
func (c core) With(fields []zapcore.Field) zapcore.Core {
	convertToECSFields(fields)
	return &core{c.Core.With(fields)}
}

// Check verifies whether or not the provided entry should be logged,
// by comparing the log level with the configured log level in the core.
// If it should be logged the core is added to the returned entry.
func (c core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

// Write converts error fields into ECS compliant errors
// before serializing the entry and fields.
func (c core) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	convertToECSFields(fields)
	fields = append(fields, zap.String("ecs.version", version))
	return c.Core.Write(ent, fields)
}

func convertToECSFields(fields []zapcore.Field) {
	for i, f := range fields {
		if f.Type == zapcore.ErrorType {
			fields[i] = zapcore.Field{Key: "error",
				Type:      zapcore.ObjectMarshalerType,
				Interface: internal.NewError(f.Interface.(error)),
			}
		}
	}
}
