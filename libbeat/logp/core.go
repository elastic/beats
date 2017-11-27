package logp

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/beats/libbeat/common/file"
	"github.com/elastic/beats/libbeat/paths"
)

var (
	root   unsafe.Pointer
	global unsafe.Pointer
	logs   *observer.ObservedLogs
)

func init() {
	setLogger(zap.NewNop())
}

func DevelopmentSetup() error {
	logger, err := zap.NewDevelopment()
	if err == nil {
		setLogger(logger)
	}
	return err
}

func MillisecondsDurationEncoder(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendFloat64(float64(d) / float64(time.Millisecond))
}

func CustomSetup(c Config) error {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: MillisecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if c.JSON {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	var level zapcore.Level
	switch strings.ToLower(c.Level) {
	case "critical", "error":
		level = zapcore.ErrorLevel
	case "warn":
		level = zapcore.WarnLevel
	case "info":
		level = zapcore.InfoLevel
	case "debug":
		level = zapcore.DebugLevel
	default:
		level = zapcore.InfoLevel
	}

	var cores []zapcore.Core
	if c.ToStderr {
		stderr := zapcore.Lock(os.Stderr)
		cores = append(cores, zapcore.NewCore(encoder, stderr, level))
	}
	if c.ToFiles {
		logFile := paths.Resolve(paths.Logs, filepath.Join(c.Files.Path, c.Files.Name))
		fileRotator, err := file.NewFileRotator(logFile,
			file.MaxSizeBytes(uint(c.Files.RotateEveryBytes)),
			file.MaxBackups(uint(c.Files.KeepFiles)),
			file.Permissions(os.FileMode(c.Files.Permissions)),
		)
		if err != nil {
			return err
		}
		cores = append(cores, zapcore.NewCore(encoder, fileRotator, level))
	}
	if c.ToStderr {
		syslog, err := NewSyslog(encoder, level)
		if err != nil {
			return err
		}
		cores = append(cores, syslog)
	}

	core := zapcore.NewTee(cores...)
	if len(c.Selectors) > 0 {
		m := make(map[string]struct{}, len(c.Selectors))
		for _, s := range c.Selectors {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if s == "*" {
				m = nil
				break
			}
			m[s] = struct{}{}
		}
		core = SelectiveWrapper(core, m)
	}

	logger := zap.New(core)
	setLogger(logger)
	return nil
}

// ObserverLogs provides the list of logs generated during the observation
// process.
func ObserverLogs() *observer.ObservedLogs {
	return logs
}

// ObserverSetup constructs a logger through the zap/zaptest/observer framework
// so that logs may be accessible in tests.
func ObserverSetup(level zapcore.Level) {
	core, observedLogs := observer.New(level)
	logs = observedLogs

	logger := zap.New(core, zap.Development())
	setLogger(logger)
}

func Sync() error {
	return rootLogger().Sync()
}

func rootLogger() *zap.Logger {
	p := atomic.LoadPointer(&root)
	return (*zap.Logger)(p)
}

func globalLogger() *zap.Logger {
	p := atomic.LoadPointer(&global)
	return (*zap.Logger)(p)
}

func setLogger(l *zap.Logger) {
	if l := rootLogger(); l != nil {
		l.Sync()
	}
	atomic.StorePointer(&root, unsafe.Pointer(l))
	atomic.StorePointer(&global, unsafe.Pointer(l.WithOptions(zap.AddCallerSkip(1))))
}

type SelectiveCore struct {
	selectors map[string]struct{}
	core      zapcore.Core
}

func SelectiveWrapper(core zapcore.Core, selectors map[string]struct{}) zapcore.Core {
	if len(selectors) == 0 {
		return core
	}
	return &SelectiveCore{selectors: selectors, core: core}
}

// Enabled returns whether a given logging level is enabled when logging a
// message.
func (c *SelectiveCore) Enabled(level Level) bool {
	return c.core.Enabled(level)
}

// With adds structured context to the Core.
func (c *SelectiveCore) With(fields []Field) zapcore.Core {
	return SelectiveWrapper(c.core.With(fields), c.selectors)
}

// Check determines whether the supplied Entry should be logged (using the
// embedded LevelEnabler and possibly some extra logic). If the entry
// should be logged, the Core adds itself to the CheckedEntry and returns
// the result.
//
// Callers must use Check before calling Write.
func (c *SelectiveCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		if ent.Level == zapcore.DebugLevel && len(c.selectors) > 0 {
			if _, enabled := c.selectors[ent.LoggerName]; enabled {
				return ce.AddCore(ent, c)
			}
			return ce
		}

		return ce.AddCore(ent, c)
	}
	return ce
}

// Write serializes the Entry and any Fields supplied at the log site and
// writes them to their destination.
//
// If called, Write should always log the Entry and Fields; it should not
// replicate the logic of Check.
func (c *SelectiveCore) Write(ent zapcore.Entry, fields []Field) error {
	return c.core.Write(ent, fields)
}

// Sync flushes buffered logs (if any).
func (c *SelectiveCore) Sync() error {
	return c.core.Sync()
}
