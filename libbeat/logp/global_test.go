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

//go:build !nologpglobal
// +build !nologpglobal

package logp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

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

func TestRecover(t *testing.T) {
	const recoveryExplanation = "Something went wrong"
	const cause = "unexpected condition"

	DevelopmentSetup(ToObserverOutput())

	defer func() {
		logs := ObserverLogs().TakeAll()
		if assert.Len(t, logs, 1) {
			log := logs[0]
			assert.Equal(t, zap.ErrorLevel, log.Level)
			assert.Equal(t, "logp/global_test.go",
				strings.Split(log.Caller.TrimmedPath(), ":")[0])
			assert.Contains(t, log.Message, recoveryExplanation+
				". Recovering, but please report this.")
			assert.Contains(t, log.ContextMap(), "panic")
		}
	}()

	defer Recover(recoveryExplanation)
	panic(cause)
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
