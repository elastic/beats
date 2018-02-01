package logp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestLogger(t *testing.T) {
	exerciseLogger := func() {
		Info("unnamed global logger")

		log := NewLogger("example")
		log.Info("some message")
		log.Infof("some message with parameter x=%v, y=%v", 1, 2)
		log.Infow("some message", "x", 1, "y", 2)
		log.Infow("some message", Int("x", 1))
		log.Infow("some message with namespaced args", Namespace("metrics"), "x", 1, "y", 1)
		log.Infow("", "empty_message", true)

		// Add context.
		log.With("x", 1, "y", 2).Warn("logger with context")

		someStruct := struct {
			X int `json:"x"`
			Y int `json:"y"`
		}{1, 2}
		log.Infow("some message with struct value", "metrics", someStruct)
	}

	TestingSetup()
	exerciseLogger()
	TestingSetup(AsJSON())
	exerciseLogger()
}

func TestLoggerSelectors(t *testing.T) {
	if err := DevelopmentSetup(WithSelectors("good"), ToObserverOutput()); err != nil {
		t.Fatal(err)
	}

	good := NewLogger("good")
	bad := NewLogger("bad")

	good.Debug("is logged")
	logs := ObserverLogs().TakeAll()
	assert.Len(t, logs, 1)

	// Selectors only apply to debug level logs.
	bad.Debug("not logged")
	logs = ObserverLogs().TakeAll()
	assert.Len(t, logs, 0)

	bad.Info("is also logged")
	logs = ObserverLogs().TakeAll()
	assert.Len(t, logs, 1)
}

func TestGlobalLoggerLevel(t *testing.T) {
	if err := DevelopmentSetup(ToObserverOutput()); err != nil {
		t.Fatal(err)
	}

	const loggerName = "tester"

	Debug(loggerName, "debug")
	logs := ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, zap.DebugLevel, logs[0].Level)
		assert.Equal(t, loggerName, logs[0].LoggerName)
		assert.Equal(t, "debug", logs[0].Message)
	}

	Info("info")
	logs = ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, zap.InfoLevel, logs[0].Level)
		assert.Equal(t, "", logs[0].LoggerName)
		assert.Equal(t, "info", logs[0].Message)
	}

	Warn("warning")
	logs = ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, zap.WarnLevel, logs[0].Level)
		assert.Equal(t, "", logs[0].LoggerName)
		assert.Equal(t, "warning", logs[0].Message)
	}

	Err("error")
	logs = ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, zap.ErrorLevel, logs[0].Level)
		assert.Equal(t, "", logs[0].LoggerName)
		assert.Equal(t, "error", logs[0].Message)
	}

	Critical("critical")
	logs = ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, zap.ErrorLevel, logs[0].Level)
		assert.Equal(t, "", logs[0].LoggerName)
		assert.Equal(t, "critical", logs[0].Message)
	}
}

func TestLoggerLevel(t *testing.T) {
	if err := DevelopmentSetup(ToObserverOutput()); err != nil {
		t.Fatal(err)
	}

	const loggerName = "tester"
	logger := NewLogger(loggerName)

	logger.Debug("debug")
	logs := ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, zap.DebugLevel, logs[0].Level)
		assert.Equal(t, loggerName, logs[0].LoggerName)
		assert.Equal(t, "debug", logs[0].Message)
	}

	logger.Info("info")
	logs = ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, zap.InfoLevel, logs[0].Level)
		assert.Equal(t, loggerName, logs[0].LoggerName)
		assert.Equal(t, "info", logs[0].Message)
	}

	logger.Warn("warn")
	logs = ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, zap.WarnLevel, logs[0].Level)
		assert.Equal(t, loggerName, logs[0].LoggerName)
		assert.Equal(t, "warn", logs[0].Message)
	}

	logger.Error("error")
	logs = ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, zap.ErrorLevel, logs[0].Level)
		assert.Equal(t, loggerName, logs[0].LoggerName)
		assert.Equal(t, "error", logs[0].Message)
	}
}

func TestRecover(t *testing.T) {
	const recoveryExplanation = "Something went wrong"
	const cause = "unexpected condition"

	DevelopmentSetup(ToObserverOutput())

	defer func() {
		logs := ObserverLogs().TakeAll()
		if assert.Len(t, logs, 1) {
			log := logs[0]
			assert.Equal(t, zap.ErrorLevel, log.Level)
			assert.Equal(t, "logp/core_test.go",
				strings.Split(log.Caller.TrimmedPath(), ":")[0])
			assert.Contains(t, log.Message, recoveryExplanation+
				". Recovering, but please report this.")
			assert.Contains(t, log.ContextMap(), "panic")
		}
	}()

	defer Recover(recoveryExplanation)
	panic(cause)
}

func TestHasSelector(t *testing.T) {
	DevelopmentSetup(WithSelectors("*", "config"))
	assert.True(t, HasSelector("config"))
	assert.False(t, HasSelector("publish"))
}

func TestIsDebug(t *testing.T) {
	DevelopmentSetup()
	assert.True(t, IsDebug("all"))

	DevelopmentSetup(WithSelectors("*"))
	assert.True(t, IsDebug("all"))

	DevelopmentSetup(WithSelectors("only_this"))
	assert.False(t, IsDebug("all"))
	assert.True(t, IsDebug("only_this"))
}
