package logp

import (
	"time"

	"go.uber.org/zap/zapcore"
)

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
	return zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: millisecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}
}

func consoleEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: millisecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     bracketedNameEncoder,
	}
}

func syslogEncoderConfig() zapcore.EncoderConfig {
	c := consoleEncoderConfig()
	// Time is added by syslog.
	c.TimeKey = ""
	return c
}

func millisecondsDurationEncoder(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendFloat64(float64(d) / float64(time.Millisecond))
}

func bracketedNameEncoder(loggerName string, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + loggerName + "]")
}
