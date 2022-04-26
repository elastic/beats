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

package logger

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestLogBrowserRun(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	SetLogger(logp.NewLogger("t", zap.WrapCore(func(in zapcore.Core) zapcore.Core {
		return zapcore.NewTee(in, core)
	})))

	durationUs := int64(5000 * time.Microsecond)
	durationMs := time.Duration(durationUs * int64(time.Microsecond)).Milliseconds()
	steps := 1337

	fields := common.MapStr{
		"monitor.id":          "b0",
		"monitor.duration.us": durationUs,
		"monitor.type":        "browser",
	}

	event := beat.Event{Fields: fields}

	LogBrowserRun(&event, steps)

	observedEntries := observed.All()
	require.Len(t, observedEntries, 1)
	assert.Equal(t, "Browser monitor summary ready", observedEntries[0].Message)

	expectedMonitor := NewMonitorRunInfo("b0", "browser", durationMs)
	expectedMonitor.Steps = &steps
	assert.ElementsMatch(t, []zap.Field{
		logp.Any("event", map[string]string{"action": ActionMonitorRun}),
		logp.Any("monitor", &expectedMonitor),
	}, observedEntries[0].Context)
}

func TestLogLightweightRun(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	SetLogger(logp.NewLogger("t", zap.WrapCore(func(in zapcore.Core) zapcore.Core {
		return zapcore.NewTee(in, core)
	})))

	durationUs := int64(5000 * time.Microsecond)
	durationMs := time.Duration(durationUs * int64(time.Microsecond)).Milliseconds()

	fields := common.MapStr{
		"monitor.id":          "h0",
		"monitor.duration.us": durationUs,
		"monitor.type":        "http",
	}

	event := beat.Event{Fields: fields}

	LogLightweightRun(&event)

	observedEntries := observed.All()
	require.Len(t, observedEntries, 1)
	assert.Equal(t, "Lightweight monitor finished.", observedEntries[0].Message)

	expectedMonitor := NewMonitorRunInfo("h0", "http", durationMs)
	assert.ElementsMatch(t, []zap.Field{
		logp.Any("event", map[string]string{"action": ActionMonitorRun}),
		logp.Any("monitor", &expectedMonitor),
	}, observedEntries[0].Context)
}
