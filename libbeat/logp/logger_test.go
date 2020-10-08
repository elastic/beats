package logp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerWithOptions(t *testing.T) {
	core1, observed1 := observer.New(zapcore.DebugLevel)
	core2, observed2 := observer.New(zapcore.DebugLevel)

	logger1 := NewLogger("bo", zap.WrapCore(func(in zapcore.Core) zapcore.Core {
		return zapcore.NewTee(in, core1)
	}))
	logger2 := logger1.WithOptions(zap.WrapCore(func(in zapcore.Core) zapcore.Core {
		return zapcore.NewTee(in, core2)
	}))

	logger1.Info("hello logger1")             // should just go to the first observer
	logger2.Info("hello logger1 and logger2") // should go to both observers

	assert.Equal(t, []observer.LoggedEntry{{
		Context: []zapcore.Field{},
		Entry: zapcore.Entry{
			Level:      zapcore.InfoLevel,
			LoggerName: "bo",
			Message:    "hello logger1",
		},
	}, {
		Context: []zapcore.Field{},
		Entry: zapcore.Entry{
			Level:      zapcore.InfoLevel,
			LoggerName: "bo",
			Message:    "hello logger1 and logger2",
		},
	}}, observed1.AllUntimed())

	assert.Equal(t, []observer.LoggedEntry{{
		Context: []zapcore.Field{},
		Entry: zapcore.Entry{
			Level:      zapcore.InfoLevel,
			LoggerName: "bo",
			Message:    "hello logger1 and logger2",
		},
	}}, observed2.AllUntimed())
}
