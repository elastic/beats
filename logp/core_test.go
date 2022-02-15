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
	"io/ioutil"
	golog "log"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestLogger(t *testing.T) {
	exerciseLogger := func() {
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
	TestingSetup()
	exerciseLogger()
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

func TestL(t *testing.T) {
	if err := DevelopmentSetup(ToObserverOutput()); err != nil {
		t.Fatal(err)
	}

	L().Infow("infow", "rate", 2)
	logs := ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		log := logs[0]
		assert.Equal(t, zap.InfoLevel, log.Level)
		assert.Equal(t, "", log.LoggerName)
		assert.Equal(t, "infow", log.Message)
		assert.Contains(t, log.ContextMap(), "rate")
	}

	const loggerName = "tester"
	L().Named(loggerName).Warnf("warning %d", 1)
	logs = ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		log := logs[0]
		assert.Equal(t, zap.WarnLevel, log.Level)
		assert.Equal(t, loggerName, log.LoggerName)
		assert.Equal(t, "warning 1", log.Message)
	}
}

func TestDebugAllStdoutEnablesDefaultGoLogger(t *testing.T) {
	DevelopmentSetup(WithSelectors("*"))
	assert.Equal(t, _defaultGoLog, golog.Writer())

	DevelopmentSetup(WithSelectors("stdlog"))
	assert.Equal(t, _defaultGoLog, golog.Writer())

	DevelopmentSetup(WithSelectors("*", "stdlog"))
	assert.Equal(t, _defaultGoLog, golog.Writer())

	DevelopmentSetup(WithSelectors("other"))
	assert.Equal(t, ioutil.Discard, golog.Writer())
}

func TestNotDebugAllStdoutDisablesDefaultGoLogger(t *testing.T) {
	DevelopmentSetup(WithSelectors("*"), WithLevel(InfoLevel))
	assert.Equal(t, ioutil.Discard, golog.Writer())

	DevelopmentSetup(WithSelectors("stdlog"), WithLevel(InfoLevel))
	assert.Equal(t, ioutil.Discard, golog.Writer())

	DevelopmentSetup(WithSelectors("*", "stdlog"), WithLevel(InfoLevel))
	assert.Equal(t, ioutil.Discard, golog.Writer())

	DevelopmentSetup(WithSelectors("other"), WithLevel(InfoLevel))
	assert.Equal(t, ioutil.Discard, golog.Writer())
}

func TestLoggingECSFields(t *testing.T) {
	cfg := Config{
		Beat:        "beat1",
		Level:       DebugLevel,
		development: true,
		Files: FileConfig{
			Name: "beat1",
		},
	}
	ToObserverOutput()(&cfg)
	Configure(cfg)

	logger := NewLogger("tester")

	logger.Debug("debug")
	logs := ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		if assert.Len(t, logs[0].Context, 1) {
			assert.Equal(t, "service.name", logs[0].Context[0].Key)
			assert.Equal(t, "beat1", logs[0].Context[0].String)
		}
	}
}
