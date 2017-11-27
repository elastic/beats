package logp

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	Field  = zapcore.Field
	Level  = zapcore.Level
	Option = zap.Option
)

// Field types for structured logging. Most fields are lazily marshaled so it
// is inexpensive to add fields to disabled log statements.
var (
	Any         = zap.Any
	Array       = zap.Array
	Binary      = zap.Binary
	Bool        = zap.Bool
	Bools       = zap.Bools
	ByteString  = zap.ByteString
	ByteStrings = zap.ByteStrings
	Complex64   = zap.Complex64
	Complex64s  = zap.Complex64s
	Complex128  = zap.Complex128
	Complex128s = zap.Complex128s
	Duration    = zap.Duration
	Durations   = zap.Durations
	Error       = zap.Error
	Errors      = zap.Errors
	Float32     = zap.Float32
	Float32s    = zap.Float32s
	Float64     = zap.Float64
	Float64s    = zap.Float64s
	Int         = zap.Int
	Ints        = zap.Ints
	Int8        = zap.Int8
	Int8s       = zap.Int8s
	Int16       = zap.Int16
	Int16s      = zap.Int16s
	Int32       = zap.Int32
	Int32s      = zap.Int32s
	Int64       = zap.Int64
	Int64s      = zap.Int64s
	Namespace   = zap.Namespace
	Reflect     = zap.Reflect
	Stack       = zap.Reflect
	String      = zap.String
	Stringer    = zap.Stringer
	Strings     = zap.Strings
	Time        = zap.Time
	Times       = zap.Times
	Uint        = zap.Uint
	Uints       = zap.Uints
	Uint8       = zap.Uint8
	Uint8s      = zap.Uint8s
	Uint16      = zap.Uint16
	Uint16s     = zap.Uint16s
	Uint32      = zap.Uint32
	Uint32s     = zap.Uint32s
	Uint64      = zap.Uint64
	Uint64s     = zap.Uint64s
	Uintptr     = zap.Uintptr
	Uintptrs    = zap.Uintptrs
)

type SimpleLogger struct {
	log   *zap.Logger
	sugar *zap.SugaredLogger
}

func NewSimpleLogger(selector string, options ...Option) *SimpleLogger {
	log := rootLogger().
		WithOptions(zap.AddCallerSkip(1)).
		WithOptions(options...).
		Named(selector)
	return &SimpleLogger{log: log, sugar: log.Sugar()}
}

// Sprint

func (l *SimpleLogger) Debug(args ...interface{}) {
	l.sugar.Debug(args...)
}

func (l *SimpleLogger) Info(args ...interface{}) {
	l.sugar.Info(args...)
}

func (l *SimpleLogger) Warn(args ...interface{}) {
	l.sugar.Warn(args...)
}

func (l *SimpleLogger) Error(args ...interface{}) {
	l.sugar.Error(args...)
}

func (l *SimpleLogger) Fatal(args ...interface{}) {
	l.sugar.Fatal(args...)
}

func (l *SimpleLogger) Panic(args ...interface{}) {
	l.sugar.Panic(args...)
}

func (l *SimpleLogger) DPanic(args ...interface{}) {
	l.sugar.DPanic(args...)
}

// Sprintf

func (l *SimpleLogger) Debugf(format string, args ...interface{}) {
	l.sugar.Debugf(format, args...)
}

func (l *SimpleLogger) Infof(format string, args ...interface{}) {
	l.sugar.Infof(format, args...)
}

func (l *SimpleLogger) Warnf(format string, args ...interface{}) {
	l.sugar.Warnf(format, args...)
}

func (l *SimpleLogger) Errorf(format string, args ...interface{}) {
	l.sugar.Errorf(format, args...)
}

func (l *SimpleLogger) Fatalf(format string, args ...interface{}) {
	l.sugar.Fatalf(format, args...)
}

func (l *SimpleLogger) Panicf(format string, args ...interface{}) {
	l.sugar.Panicf(format, args...)
}

func (l *SimpleLogger) DPanicf(format string, args ...interface{}) {
	l.sugar.DPanicf(format, args...)
}

// With context (reflection based)

func (l *SimpleLogger) Debugw(msg string, keysAndValues ...interface{}) {
	l.sugar.Debugw(msg, keysAndValues...)
}

func (l *SimpleLogger) Infow(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, keysAndValues...)
}

func (l *SimpleLogger) Warnw(msg string, keysAndValues ...interface{}) {
	l.sugar.Warnw(msg, keysAndValues...)
}

func (l *SimpleLogger) Errorw(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, keysAndValues...)
}

func (l *SimpleLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.sugar.Fatalw(msg, keysAndValues...)
}

func (l *SimpleLogger) Panicw(msg string, keysAndValues ...interface{}) {
	l.sugar.Panicw(msg, keysAndValues...)
}

func (l *SimpleLogger) DPanicw(msg string, keysAndValues ...interface{}) {
	l.sugar.DPanicw(msg, keysAndValues...)
}

// Strongly typed Logger (more performant)

type Logger struct {
	log *zap.Logger
}

func NewLogger(selector string, options ...Option) *Logger {
	log := rootLogger().
		WithOptions(zap.AddCallerSkip(1)).
		WithOptions(options...).
		Named(selector)
	return &Logger{log: log}
}

func (l *Logger) Check(level Level, msg string) *zapcore.CheckedEntry {
	return l.log.Check(level, msg)
}

func (l *Logger) Debug(msg string, fields ...Field) {
	l.log.Debug(msg, fields...)
}

func (l *Logger) Info(msg string, fields ...Field) {
	l.log.Info(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...Field) {
	l.log.Warn(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...Field) {
	l.log.Error(msg, fields...)
}

func (l *Logger) Fatal(msg string, fields ...Field) {
	l.log.Fatal(msg, fields...)
}

func (l *Logger) Panic(msg string, fields ...Field) {
	l.log.Panic(msg, fields...)
}

func (l *Logger) DPanic(msg string, fields ...Field) {
	l.log.DPanic(msg, fields...)
}
