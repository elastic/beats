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

	"github.com/elastic/elastic-agent-libs/logp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

// NewTestingLogger returns a testing suitable logp.Logger.
func NewTestingLogger(t testing.TB, selector string, options ...logp.LogOption) *logp.Logger {
	log := zaptest.NewLogger(t)
	log = log.Named(selector)
	options = append(options, zap.WrapCore(func(zapcore.Core) zapcore.Core {
		return log.Core()
	}))
	logger, err := logp.NewDevelopmentLogger(selector, options...)
	if err != nil {
		t.Fatal(err)
	}
	return logger
}
