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

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestScrubberLogger(t *testing.T) {
	const secret = "s3cr3t-token"
	base, logs := logptest.NewTestingLoggerWithObserver(t, "")
	logger := scrubberFor(secret).Logger(base).With("param", "p="+secret)

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

func TestScrubberLoggerKeepsLevel(t *testing.T) {
	base, logs := logptest.NewTestingLoggerWithObserver(t, "")
	logger := scrubberFor("s3cr3t-token").Logger(base)

	logger.Debug("debug s3cr3t-token")
	logger.Warn("warn s3cr3t-token")

	entries := logs.TakeAll()
	require.Len(t, entries, 2, "the scrubbing logger must keep the levels of the base logger")
	assert.Equal(t, "warn <hidden>", entries[1].Message, "the message must not have the secret")
}

func TestNilScrubberLogger(t *testing.T) {
	base := logptest.NewTestingLogger(t, "")
	var s *Scrubber
	assert.Same(t, base, s.Logger(base), "a nil Scrubber must return the logger as is")
	assert.Nil(t, scrubberFor("s3cr3t-token").Logger(nil), "a nil logger must stay nil")
}
