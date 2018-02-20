package logp

import (
	"go.uber.org/zap/zapcore"
)

var baseEncodingConfig = zapcore.EncoderConfig{
	TimeKey:        "timestamp",
	LevelKey:       "level",
	NameKey:        "logger",
	CallerKey:      "caller",
	MessageKey:     "message",
	StacktraceKey:  "stacktrace",
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    zapcore.LowercaseLevelEncoder,
	EncodeTime:     zapcore.ISO8601TimeEncoder,
	EncodeDuration: zapcore.NanosDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
	EncodeName:     zapcore.FullNameEncoder,
}

func buildEncoder(cfg Config) zapcore.Encoder {
	if cfg.JSON {
		return zapcore.NewJSONEncoder(jsonEncoderConfig())
	} else if cfg.ToSyslog {
		return zapcore.NewConsoleEncoder(syslogEncoderConfig())
	} else {
		return zapcore.NewConsoleEncoder(consoleEncoderConfig())
	}
}

func jsonEncoderConfig() zapcore.EncoderConfig {
	return baseEncodingConfig
}

func consoleEncoderConfig() zapcore.EncoderConfig {
	c := baseEncodingConfig
	c.EncodeLevel = zapcore.CapitalLevelEncoder
	c.EncodeName = bracketedNameEncoder
	return c
}

func syslogEncoderConfig() zapcore.EncoderConfig {
	c := consoleEncoderConfig()
	// Time is added by syslog.
	c.TimeKey = ""
	return c
}

func bracketedNameEncoder(loggerName string, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + loggerName + "]")
}
