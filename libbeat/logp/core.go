// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package logp

import (
	"flag"
	"io/ioutil"
	golog "log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/hashicorp/go-multierror"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.elastic.co/ecszap"

	"github.com/menderesk/beats/v7/libbeat/common/file"
	"github.com/menderesk/beats/v7/libbeat/paths"
)

var (
	_log          unsafe.Pointer // Pointer to a coreLogger. Access via atomic.LoadPointer.
	_defaultGoLog = golog.Writer()
)

func init() {
	storeLogger(&coreLogger{
		selectors:    map[string]struct{}{},
		rootLogger:   zap.NewNop(),
		globalLogger: zap.NewNop(),
		logger:       newLogger(zap.NewNop(), ""),
	})
}

type coreLogger struct {
	selectors    map[string]struct{}    // Set of enabled debug selectors.
	rootLogger   *zap.Logger            // Root logger without any options configured.
	globalLogger *zap.Logger            // Logger used by legacy global functions (e.g. logp.Info).
	logger       *Logger                // Logger that is the basis for all logp.Loggers.
	observedLogs *observer.ObservedLogs // Contains events generated while in observation mode (a testing mode).
}

// Configure configures the logp package.
func Configure(cfg Config) error {
	return ConfigureWithOutputs(cfg)
}

// ConfigureWithOutputs XXX: is used by elastic-agent only (See file: x-pack/elastic-agent/pkg/core/logger/logger.go).
// The agent requires that the output specified in the config object is configured and merged with the
// logging outputs given.
func ConfigureWithOutputs(cfg Config, outputs ...zapcore.Core) error {
	var (
		sink         zapcore.Core
		observedLogs *observer.ObservedLogs
		err          error
	)

	// Build a single output (stderr has priority if more than one are enabled).
	if cfg.toObserver {
		sink, observedLogs = observer.New(cfg.Level.ZapLevel())
	} else {
		sink, err = createLogOutput(cfg)
	}
	if err != nil {
		return errors.Wrap(err, "failed to build log output")
	}

	// Default logger is always discard, debug level below will
	// possibly re-enable it.
	golog.SetOutput(ioutil.Discard)

	// Enabled selectors when debug is enabled.
	selectors := make(map[string]struct{}, len(cfg.Selectors))
	if cfg.Level.Enabled(DebugLevel) && len(cfg.Selectors) > 0 {
		for _, sel := range cfg.Selectors {
			selectors[strings.TrimSpace(sel)] = struct{}{}
		}

		// Default to all enabled if no selectors are specified.
		if len(selectors) == 0 {
			selectors["*"] = struct{}{}
		}

		// Re-enable the default go logger output when either stdlog
		// or all selector is enabled.
		_, stdlogEnabled := selectors["stdlog"]
		_, allEnabled := selectors["*"]
		if stdlogEnabled || allEnabled {
			golog.SetOutput(_defaultGoLog)
		}

		sink = selectiveWrapper(sink, selectors)
	}

	sink = newMultiCore(append(outputs, sink)...)
	root := zap.New(sink, makeOptions(cfg)...)
	storeLogger(&coreLogger{
		selectors:    selectors,
		rootLogger:   root,
		globalLogger: root.WithOptions(zap.AddCallerSkip(1)),
		logger:       newLogger(root, ""),
		observedLogs: observedLogs,
	})
	return nil
}

func createLogOutput(cfg Config) (zapcore.Core, error) {
	switch {
	case cfg.toIODiscard:
		return makeDiscardOutput(cfg)
	case cfg.ToStderr:
		return makeStderrOutput(cfg)
	case cfg.ToSyslog:
		return makeSyslogOutput(cfg)
	case cfg.ToEventLog:
		return makeEventLogOutput(cfg)
	case cfg.ToFiles:
		return makeFileOutput(cfg)
	}

	switch cfg.environment {
	case SystemdEnvironment, ContainerEnvironment:
		return makeStderrOutput(cfg)
	case MacOSServiceEnvironment, WindowsServiceEnvironment:
		fallthrough
	default:
		return makeFileOutput(cfg)
	}
}

// DevelopmentSetup configures the logger in development mode at debug level.
// By default the output goes to stderr.
func DevelopmentSetup(options ...Option) error {
	cfg := Config{
		Level:       DebugLevel,
		ToStderr:    true,
		development: true,
		addCaller:   true,
	}
	for _, apply := range options {
		apply(&cfg)
	}
	return Configure(cfg)
}

// TestingSetup configures logging by calling DevelopmentSetup if and only if
// verbose testing is enabled (as in 'go test -v').
func TestingSetup(options ...Option) error {
	// Use the flag to avoid a dependency on the testing package.
	f := flag.Lookup("test.v")
	if f != nil && f.Value.String() == "true" {
		return DevelopmentSetup(options...)
	}
	return nil
}

// ObserverLogs provides the list of logs generated during the observation
// process.
func ObserverLogs() *observer.ObservedLogs {
	return loadLogger().observedLogs
}

// Sync flushes any buffered log entries. Applications should take care to call
// Sync before exiting.
func Sync() error {
	return loadLogger().rootLogger.Sync()
}

func makeOptions(cfg Config) []zap.Option {
	var options []zap.Option
	if cfg.addCaller {
		options = append(options, zap.AddCaller())
	}
	if cfg.development {
		options = append(options, zap.Development())
	}
	if cfg.Beat != "" {
		fields := []zap.Field{
			zap.String("service.name", cfg.Beat),
		}
		options = append(options, zap.Fields(fields...))
	}
	return options
}

func makeStderrOutput(cfg Config) (zapcore.Core, error) {
	stderr := zapcore.Lock(os.Stderr)
	return newCore(cfg, buildEncoder(cfg), stderr, cfg.Level.ZapLevel()), nil
}

func makeDiscardOutput(cfg Config) (zapcore.Core, error) {
	discard := zapcore.AddSync(ioutil.Discard)
	return newCore(cfg, buildEncoder(cfg), discard, cfg.Level.ZapLevel()), nil
}

func makeSyslogOutput(cfg Config) (zapcore.Core, error) {
	core, err := newSyslog(buildEncoder(cfg), cfg.Level.ZapLevel())
	if err != nil {
		return nil, err
	}
	return wrappedCore(cfg, core), nil
}

func makeEventLogOutput(cfg Config) (zapcore.Core, error) {
	core, err := newEventLog(cfg.Beat, buildEncoder(cfg), cfg.Level.ZapLevel())
	if err != nil {
		return nil, err
	}
	return wrappedCore(cfg, core), nil
}

func makeFileOutput(cfg Config) (zapcore.Core, error) {
	filename := paths.Resolve(paths.Logs, filepath.Join(cfg.Files.Path, cfg.LogFilename()))

	rotator, err := file.NewFileRotator(filename,
		file.MaxSizeBytes(cfg.Files.MaxSize),
		file.MaxBackups(cfg.Files.MaxBackups),
		file.Permissions(os.FileMode(cfg.Files.Permissions)),
		file.Interval(cfg.Files.Interval),
		file.RotateOnStartup(cfg.Files.RotateOnStartup),
		file.RedirectStderr(cfg.Files.RedirectStderr),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create file rotator")
	}

	return newCore(cfg, buildEncoder(cfg), rotator, cfg.Level.ZapLevel()), nil
}

func newCore(cfg Config, enc zapcore.Encoder, ws zapcore.WriteSyncer, enab zapcore.LevelEnabler) zapcore.Core {
	return wrappedCore(cfg, zapcore.NewCore(enc, ws, enab))
}
func wrappedCore(cfg Config, core zapcore.Core) zapcore.Core {
	return ecszap.WrapCore(core)
}

func globalLogger() *zap.Logger {
	return loadLogger().globalLogger
}

func loadLogger() *coreLogger {
	p := atomic.LoadPointer(&_log)
	return (*coreLogger)(p)
}

func storeLogger(l *coreLogger) {
	if old := loadLogger(); old != nil {
		old.rootLogger.Sync()
	}
	atomic.StorePointer(&_log, unsafe.Pointer(l))
}

// newMultiCore creates a sink that sends to multiple cores.
func newMultiCore(cores ...zapcore.Core) zapcore.Core {
	return &multiCore{cores}
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
