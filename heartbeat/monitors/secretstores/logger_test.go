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

package secretstores

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

func TestRedactorLogger(t *testing.T) {
	const secret = "s3cr3t-token"
	base, logs := logptest.NewTestingLoggerWithObserver(t, "")
	logger := redactorFor(secret).Logger(base).With("param", "p="+secret)

	logger.Infow("running --params {\"token\":\""+secret+"\"}",
		"line", "stdout: "+secret,
		zap.Error(errors.New("failed with "+secret)),
		"count", 3,
	)

	entries := logs.TakeAll()
	require.Len(t, entries, 1, "the log line must be written")
	e := entries[0]
	assert.Equal(t, `running --params {"token":"<hidden>"}`, e.Message, "the message must not have the secret")
	fields := e.ContextMap()
	assert.Equal(t, "p=<hidden>", fields["param"], "the With field must not have the secret")
	assert.Equal(t, "stdout: <hidden>", fields["line"], "the string field must not have the secret")
	assert.Equal(t, "failed with <hidden>", fields["error"], "the error field must not have the secret")
	assert.EqualValues(t, 3, fields["count"], "other fields must be kept")
}

func TestRedactorLoggerKeepsLevel(t *testing.T) {
	base, logs := logptest.NewTestingLoggerWithObserver(t, "")
	logger := redactorFor("s3cr3t-token").Logger(base)

	logger.Debug("debug s3cr3t-token")
	logger.Warn("warn s3cr3t-token")

	entries := logs.TakeAll()
	require.Len(t, entries, 2, "the scrubbing logger must keep the levels of the base logger")
	assert.Equal(t, "warn <hidden>", entries[1].Message, "the message must not have the secret")
}

// selectorCore drops debug entries of other loggers than selector in Check,
// as logp does for logging.selectors.
type selectorCore struct {
	zapcore.Core
	selector string
}

func (c selectorCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if e.Level == zapcore.DebugLevel && e.LoggerName != c.selector {
		return ce
	}
	return c.Core.Check(e, ce)
}

func TestRedactorLoggerKeepsSelectors(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	base, err := logp.NewZapLogger(zap.New(selectorCore{Core: core, selector: "enabled"}))
	require.NoError(t, err, "the base logger must be created")
	logger := redactorFor("s3cr3t-token").Logger(base)

	logger.Named("other").Debug("other s3cr3t-token")
	logger.Named("enabled").Debug("enabled s3cr3t-token")

	entries := logs.TakeAll()
	require.Len(t, entries, 1, "debug entries of other selectors must be dropped")
	assert.Equal(t, "enabled <hidden>", entries[0].Message, "the message must not have the secret")
}

func TestNilRedactorLogger(t *testing.T) {
	base := logptest.NewTestingLogger(t, "")
	var r *Redactor
	assert.Same(t, base, r.Logger(base), "a nil Redactor must return the logger as is")
	assert.Nil(t, redactorFor("s3cr3t-token").Logger(nil), "a nil logger must stay nil")
}

func TestRedactorLoggerWrapsOnce(t *testing.T) {
	base := logptest.NewTestingLogger(t, "")
	r := redactorFor("s3cr3t-token")
	logger := r.Logger(base)
	assert.Same(t, logger, r.Logger(logger), "a logger the redactor already scrubs must be returned as is")
	assert.NotSame(t, logger, NewRedactor().Logger(logger), "another redactor must wrap it")
}
