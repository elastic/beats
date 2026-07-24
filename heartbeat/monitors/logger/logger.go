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
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/summarizer/jobsummary"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
)

var mtx = sync.Mutex{}
var eventLogger *logp.Logger = nil

const ActionMonitorRun = "monitor.run"

const META_STEP_COUNT = "__HEARTBEAT_STEP_COUNT__"

type DurationLoggable struct {
	Mills int64 `json:"ms"`
}

type NetworkInfo map[string]interface{}

type MonitorRunInfo struct {
	MonitorID   string      `json:"id"`
	Type        string      `json:"type"`
	Duration    int64       `json:"-"`
	Steps       *int        `json:"steps,omitempty"`
	Status      string      `json:"status"`
	Attempt     int         `json:"attempt"`
	NetworkInfo NetworkInfo `json:"network_info,omitempty"`
}

func (m *MonitorRunInfo) MarshalJSON() ([]byte, error) {
	// Alias to avoid recursing on marshal
	type MonitorRunInfoAlias MonitorRunInfo
	return json.Marshal(&struct {
		*MonitorRunInfoAlias
		DurationMS DurationLoggable `json:"duration"`
	}{
		MonitorRunInfoAlias: (*MonitorRunInfoAlias)(m),
		DurationMS:          DurationLoggable{Mills: time.Duration(m.Duration * int64(time.Microsecond)).Milliseconds()},
	})
}

func SetLogger(l *logp.Logger) *logp.Logger {
	eventLogger = l
	return eventLogger
}

func getLogger() *logp.Logger {
	mtx.Lock()
	defer mtx.Unlock()

	if eventLogger == nil {
		// Fall back to the global logger if SetLogger was never called (early/misconfigured paths).
		return SetLogger(logp.L()) //nolint:forbidigo // intentional fallback before SetLogger is called
	}

	return eventLogger
}

func extractRunInfo(event *beat.Event) (*MonitorRunInfo, error) {
	errors := []error{}
	monitorIDIface, err := event.GetValue("monitor.id")
	if err != nil {
		errors = append(errors, fmt.Errorf("could not extract monitor.id: %w", err))
	}

	durationUsIface, err := event.GetValue("monitor.duration.us")
	if err != nil {
		durationUsIface = int64(0)
	}

	monTypeIface, err := event.GetValue("monitor.type")
	if err != nil {
		errors = append(errors, fmt.Errorf("could not extract monitor.type: %w", err))
	}

	statusIface, err := event.GetValue("monitor.status")
	if err != nil {
		errors = append(errors, fmt.Errorf("could not extract monitor.status: %w", err))
	}

	jsIface, err := event.GetValue("summary")
	var attempt int
	if err != nil {
		errors = append(errors, fmt.Errorf("could not extract summary to add attempt info: %w", err))
	} else {
		js, ok := jsIface.(*jobsummary.JobSummary)
		if ok && js != nil {
			attempt = int(js.Attempt)
		}
	}

	monitorID, ok := monitorIDIface.(string)
	if !ok {
		errors = append(errors, fmt.Errorf("monitor.id is not a string, got %T", monitorIDIface))
	}
	monType, ok := monTypeIface.(string)
	if !ok {
		errors = append(errors, fmt.Errorf("monitor.type is not a string, got %T", monTypeIface))
	}
	durationUs, ok := durationUsIface.(int64)
	if !ok {
		errors = append(errors, fmt.Errorf("monitor.duration.us is not an int64, got %T", durationUsIface))
	}
	status, ok := statusIface.(string)
	if !ok {
		errors = append(errors, fmt.Errorf("monitor.status is not a string, got %T", statusIface))
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("logErrors: %+v", errors)
	}

	monitor := MonitorRunInfo{
		MonitorID:   monitorID,
		Type:        monType,
		Duration:    durationUs,
		Status:      status,
		Attempt:     attempt,
		NetworkInfo: extractNetworkInfo(event, monType),
	}

	sc, _ := event.Meta.GetValue(META_STEP_COUNT)
	stepCount, ok := sc.(int)
	if ok {
		monitor.Steps = &stepCount
	}

	return &monitor, nil
}

func extractNetworkInfo(event *beat.Event, monitorType string) NetworkInfo {
	// Synthetics-driven monitors (browser, api) emit their own network_info via synthexec.
	if stdfields.IsSyntheticsType(monitorType) {
		return nil
	}

	fields := []string{
		"resolve.ip", "resolve.rtt.us", "tls.rtt.handshake.us", "icmp.rtt.us",
		"tcp.rtt.connect.us", "tcp.rtt.validate.us", "http.rtt.content.us", "http.rtt.validate.us",
		"http.rtt.validate_body.us", "http.rtt.write_request.us", "http.rtt.response_header.us",
		"http.rtt.total.us", "socks5.rtt.connect.us",
	}
	networkInfo := make(NetworkInfo)
	for _, field := range fields {
		value, err := event.GetValue(field)
		if err == nil && value != nil {
			networkInfo[field] = value
		}
	}

	return networkInfo
}

func LogRun(event *beat.Event) {
	monitor, err := extractRunInfo(event)
	if err != nil {
		getLogger().Error(fmt.Errorf("error gathering information to log event: %w", err))
		return
	}

	getLogger().Infow(
		"Monitor finished",
		logp.Any("event", map[string]string{"action": ActionMonitorRun}),
		logp.Any("monitor", monitor),
	)
}
