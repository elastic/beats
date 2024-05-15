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
	"errors"
	"flag"
	"fmt"
	"io"
	golog "log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"unsafe"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.elastic.co/ecszap"

	"github.com/elastic/elastic-agent-libs/file"
	"github.com/elastic/elastic-agent-libs/paths"
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
		level:        zap.NewAtomicLevel(),
		logger:       newLogger(zap.NewNop(), ""),
	})
}

type coreLogger struct {
	selectors    map[string]struct{}    // Set of enabled debug selectors.
	rootLogger   *zap.Logger            // Root logger without any options configured.
	globalLogger *zap.Logger            // Logger used by legacy global functions (e.g. logp.Info).
	logger       *Logger                // Logger that is the basis for all logp.Loggers.
	level        zap.AtomicLevel        // The minimum level being printed
	observedLogs *observer.ObservedLogs // Contains events generated while in observation mode (a testing mode).
}

// Configure configures the logp package.
func Configure(cfg Config) error {
	return ConfigureWithOutputs(cfg)
}

func createSink(defaultLoggerCfg Config, outputs ...zapcore.Core) (zapcore.Core, zap.AtomicLevel, *observer.ObservedLogs, map[string]struct{}, error) {
	var (
		sink         zapcore.Core
		observedLogs *observer.ObservedLogs
		err          error
		level        zap.AtomicLevel
	)

	level = zap.NewAtomicLevelAt(defaultLoggerCfg.Level.ZapLevel())
	// Build a single output (stderr has priority if more than one are enabled).
	if defaultLoggerCfg.toObserver {
		sink, observedLogs = observer.New(level)
	} else {
		sink, err = createLogOutput(defaultLoggerCfg, level)
	}
	if err != nil {
		return nil, level, nil, nil, fmt.Errorf("failed to build log output: %w", err)
	}

	// Default logger is always discard, debug level below will
	// possibly re-enable it.
	golog.SetOutput(io.Discard)

	// Enabled selectors when debug is enabled.
	selectors := make(map[string]struct{}, len(defaultLoggerCfg.Selectors))
	if defaultLoggerCfg.Level.Enabled(DebugLevel) && len(defaultLoggerCfg.Selectors) > 0 {
		for _, sel := range defaultLoggerCfg.Selectors {
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

	return sink, level, observedLogs, selectors, err
}

// ConfigureWithOutputs configures the global logger to use an output created
// from `defaultLoggerCfg` and all the outputs passed by `outputs`.
// This function needs to be exported because it's used by `logp/configure`
func ConfigureWithOutputs(defaultLoggerCfg Config, outputs ...zapcore.Core) error {
	sink, level, observedLogs, selectors, err := createSink(defaultLoggerCfg, outputs...)
	if err != nil {
		return err
	}
	root := zap.New(sink, makeOptions(defaultLoggerCfg)...)
	storeLogger(&coreLogger{
		selectors:    selectors,
		rootLogger:   root,
		globalLogger: root.WithOptions(zap.AddCallerSkip(1)),
		logger:       newLogger(root, ""),
		level:        level,
		observedLogs: observedLogs,
	})
	return nil
}

// ConfigureWithTypedOutput configures the global logger to use typed outputs.
//
// If a log entry matches the defined key/value, this entry is logged using the
// core generated from `typedLoggerCfg`, otherwise it will be logged by all
// cores in `outputs` and the one generated from `defaultLoggerCfg`.
// Arguments:
//   - `defaultLoggerCfg` is used to create a new core that will be the default
//     output from the logger
//   - `typedLoggerCfg` is used to create a new output that will only be used
//     when the log entry matches `entry[logKey] = kind`
//   - `key` is the key the typed logger will look at
//   - `value` is the value compared against the `logKey` entry
//   - `outputs` is a list of cores that will be added together with the core
//     generated by `defaultLoggerCfg` as the default output for the loggger.
//
// If `defaultLoggerCfg.toObserver` is true, then `typedLoggerCfg` is ignored
// and a single sink is used so all logs can be observed.
func ConfigureWithTypedOutput(defaultLoggerCfg, typedLoggerCfg Config, key, value string, outputs ...zapcore.Core) error {
	sink, level, observedLogs, selectors, err := createSink(defaultLoggerCfg, outputs...)
	if err != nil {
		return err
	}

	var typedCore zapcore.Core
	if defaultLoggerCfg.toObserver {
		typedCore = sink
	} else {
		typedCore, err = createLogOutput(typedLoggerCfg, level)
	}
	if err != nil {
		return fmt.Errorf("could not create typed logger output: %w", err)
	}

	sink = &typedLoggerCore{
		defaultCore: sink,
		typedCore:   typedCore,
		key:         key,
		value:       value,
	}

	sink = selectiveWrapper(sink, selectors)

	root := zap.New(sink, makeOptions(defaultLoggerCfg)...)
	storeLogger(&coreLogger{
		selectors:    selectors,
		rootLogger:   root,
		globalLogger: root.WithOptions(zap.AddCallerSkip(1)),
		logger:       newLogger(root, ""),
		level:        level,
		observedLogs: observedLogs,
	})
	return nil
}

func createLogOutput(cfg Config, enab zapcore.LevelEnabler) (zapcore.Core, error) {
	switch {
	case cfg.toIODiscard:
		return makeDiscardOutput(cfg, enab)
	case cfg.ToStderr:
		return makeStderrOutput(cfg, enab)
	case cfg.ToSyslog:
		return makeSyslogOutput(cfg, enab)
	case cfg.ToEventLog:
		return makeEventLogOutput(cfg, enab)
	case cfg.ToFiles:
		return makeFileOutput(cfg, enab)
	}

	switch cfg.environment {
	case SystemdEnvironment, ContainerEnvironment:
		return makeStderrOutput(cfg, enab)
	case MacOSServiceEnvironment, WindowsServiceEnvironment:
		return makeFileOutput(cfg, enab)
	default:
		return zapcore.NewNopCore(), nil
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

func makeStderrOutput(cfg Config, enab zapcore.LevelEnabler) (zapcore.Core, error) {
	stderr := zapcore.Lock(os.Stderr)
	return newCore(buildEncoder(cfg), stderr, enab), nil
}

func makeDiscardOutput(cfg Config, enab zapcore.LevelEnabler) (zapcore.Core, error) {
	discard := zapcore.AddSync(io.Discard)
	return newCore(buildEncoder(cfg), discard, enab), nil
}

func makeSyslogOutput(cfg Config, enab zapcore.LevelEnabler) (zapcore.Core, error) {
	core, err := newSyslog(buildEncoder(cfg), enab)
	if err != nil {
		return nil, err
	}
	return wrappedCore(core), nil
}

func makeEventLogOutput(cfg Config, enab zapcore.LevelEnabler) (zapcore.Core, error) {
	core, err := newEventLog(cfg.Beat, buildEncoder(cfg), enab)
	// nolint: staticcheck,nolintlint // the implementation is OS-specific and some implementations always return errors
	if err != nil {
		return nil, err
	}
	return wrappedCore(core), nil
}

func makeFileOutput(cfg Config, enab zapcore.LevelEnabler) (zapcore.Core, error) {
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
		return nil, fmt.Errorf("failed to create file rotator: %w", err)
	}

	return newCore(buildEncoder(cfg), rotator, enab), nil
}

func newCore(enc zapcore.Encoder, ws zapcore.WriteSyncer, enab zapcore.LevelEnabler) zapcore.Core {
	return wrappedCore(zapcore.NewCore(enc, ws, enab))
}
func wrappedCore(core zapcore.Core) zapcore.Core {
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
		_ = old.rootLogger.Sync()
	}
	atomic.StorePointer(&_log, unsafe.Pointer(l))
}

func SetLevel(lvl zapcore.Level) {
	loadLogger().level.SetLevel(lvl)
}

func GetLevel() zapcore.Level {
	return loadLogger().level.Level()
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
	var errs []error
	for _, core := range m.cores {
		if err := core.Write(entry, fields); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Sync syncs each core.
func (m multiCore) Sync() error {
	var errs []error
	for _, core := range m.cores {
		if err := core.Sync(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
