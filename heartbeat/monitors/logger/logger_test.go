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

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/summarizer/jobsummary"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func generateFakeNetworkInfo() NetworkInfo {
	networkInfo := NetworkInfo{
		// All network info available in HB documentation
		"resolve.ip":                  "192.168.1.254",
		"resolve.rtt.us":              123,
		"tls.rtt.handshake.us":        456,
		"icmp.rtt.us":                 111,
		"tcp.rtt.connect.us":          789,
		"tcp.rtt.validate.us":         1234,
		"http.rtt.content.us":         4567,
		"http.rtt.validate.us":        7890,
		"http.rtt.validate_body.us":   12345,
		"http.rtt.write_request.us":   45678,
		"http.rtt.response_header.us": 78901,
		"http.rtt.total.us":           123456,
		"socks5.rtt.connect.us":       789012,
	}

	return networkInfo
}

func TestLogRun(t *testing.T) {
	t.Run("should log the monitor completion", func(t *testing.T) {
		core, observed := observer.New(zapcore.InfoLevel)
		SetLogger(logp.NewLogger("t", zap.WrapCore(func(in zapcore.Core) zapcore.Core {
			return zapcore.NewTee(in, core)
		})))

		durationUs := int64(5000 * time.Microsecond)
		steps := 1337
		fields := mapstr.M{
			"monitor.id":          "b0",
			"monitor.duration.us": durationUs,
			"monitor.type":        "browser",
			"monitor.status":      "down",
			"summary":             jobsummary.NewJobSummary(1, 1, "abc"),
		}

		event := beat.Event{Fields: fields}
		eventext.SetMeta(&event, META_STEP_COUNT, steps)

		LogRun(&event)

		observedEntries := observed.All()
		require.Len(t, observedEntries, 1)
		assert.Equal(t, "Monitor finished", observedEntries[0].Message)

		expectedMonitor := MonitorRunInfo{
			MonitorID: "b0",
			Type:      "browser",
			Duration:  durationUs,
			Status:    "down",
			Steps:     &steps,
			Attempt:   1,
		}

		assert.ElementsMatch(t, []zap.Field{
			logp.Any("event", map[string]string{"action": ActionMonitorRun}),
			logp.Any("monitor", &expectedMonitor),
		}, observedEntries[0].Context)
	})

	t.Run("should log network information if available", func(t *testing.T) {
		core, observed := observer.New(zapcore.InfoLevel)
		SetLogger(logp.NewLogger("t", zap.WrapCore(func(in zapcore.Core) zapcore.Core {
			return zapcore.NewTee(in, core)
		})))

		durationUs := int64(5000 * time.Microsecond)
		steps := 1337
		fields := mapstr.M{
			"monitor.id":          "b0",
			"monitor.duration.us": durationUs,
			"monitor.type":        "http",
			"monitor.status":      "down",
			"summary":             jobsummary.NewJobSummary(1, 1, "abc"),
		}
		networkInfo := generateFakeNetworkInfo()
		// Add network info to the event
		for key, value := range networkInfo {
			fields[key] = value
		}

		event := beat.Event{Fields: fields}
		eventext.SetMeta(&event, META_STEP_COUNT, steps)

		LogRun(&event)

		observedEntries := observed.All()
		require.Len(t, observedEntries, 1)
		assert.Equal(t, "Monitor finished", observedEntries[0].Message)

		expectedMonitor := MonitorRunInfo{
			MonitorID:   "b0",
			Type:        "http",
			Duration:    durationUs,
			Status:      "down",
			Steps:       &steps,
			Attempt:     1,
			NetworkInfo: networkInfo,
		}

		assert.ElementsMatch(t, []zap.Field{
			logp.Any("event", map[string]string{"action": ActionMonitorRun}),
			logp.Any("monitor", &expectedMonitor),
		}, observedEntries[0].Context)
	})
}
