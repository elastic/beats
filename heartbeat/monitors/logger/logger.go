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
	"errors"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
)

var eventLogger *logp.Logger = nil

const ActionMonitorRun = "monitor.run"

type durationLoggable struct {
	Ms int64 `json:"ms"`
}

type monitorRunInfo struct {
	MonitorID string           `json:"id"`
	Type      string           `json:"type"`
	Duration  durationLoggable `json:"duration"`
	Steps     *int             `json:"steps,omitempty"`
}

func NewMonitorRunInfo(id string, t string, dMs int64) monitorRunInfo {
	return monitorRunInfo{
		MonitorID: id,
		Type:      t,
		Duration:  durationLoggable{Ms: dMs},
	}
}

func SetLogger(l *logp.Logger) *logp.Logger {
	eventLogger = l
	return eventLogger
}

func getLogger() *logp.Logger {
	if eventLogger == nil {
		return SetLogger(logp.NewLogger("heartbeat.events"))
	}

	return eventLogger
}

func extractRunInfo(event *beat.Event) (*monitorRunInfo, error) {
	monitorID, mIDErr := event.GetValue("monitor.id")
	durationUs, dErr := event.GetValue("monitor.duration.us")
	monType, tErr := event.GetValue("monitor.type")
	if mIDErr != nil || dErr != nil || tErr != nil {
		getLogger().Errorw(
			"Error gathering information to log event",
			logp.Errors("logErrors", []error{mIDErr, dErr, tErr}),
		)

		return nil, errors.New("error gathering information to log event")
	}

	durationMs := time.Duration(durationUs.(int64) * int64(time.Microsecond)).Milliseconds()
	runInfo := NewMonitorRunInfo(
		monitorID.(string),
		monType.(string),
		durationMs,
	)

	return &runInfo, nil
}

func LogBrowserRun(event *beat.Event, stepCount int) {
	monitor, err := extractRunInfo(event)
	if err != nil {
		return
	}

	monitor.Steps = &stepCount

	getLogger().Infow(
		"Browser monitor summary ready",
		logp.Any("event", map[string]string{"action": ActionMonitorRun}),
		logp.Any("monitor", monitor),
	)
}

func LogLightweightRun(event *beat.Event) {
	monitor, err := extractRunInfo(event)
	if err != nil {
		return
	}

	getLogger().Infow(
		"Lightweight monitor finished.",
		logp.Any("event", map[string]string{"action": ActionMonitorRun}),
		logp.Any("monitor", monitor),
	)
}
