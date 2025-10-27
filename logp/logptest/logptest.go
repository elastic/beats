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

package logptest

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/elastic-agent-libs/logp"
)

// NewTestingLogger returns a testing suitable logp.Logger.
func NewTestingLogger(t testing.TB, selector string, options ...logp.LogOption) *logp.Logger {
	log := zaptest.NewLogger(t, zaptest.WrapOptions(options...))
	log = log.Named(selector)

	logger, err := logp.NewZapLogger(log)
	if err != nil {
		t.Fatal(err)
	}
	return logger
}

// NewTestingLoggerWithObserver returns a testing suitable logp.Logger and an observer
func NewTestingLoggerWithObserver(t testing.TB, selector string) (*logp.Logger, *observer.ObservedLogs) {
	observedCore, observedLogs := observer.New(zapcore.DebugLevel)
	logger := NewTestingLogger(t, selector, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return observedCore
	}))

	return logger, observedLogs
}
