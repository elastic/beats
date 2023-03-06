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

type MonitorRunInfo struct {
	MonitorID string `json:"id"`
	Type      string `json:"type"`
	Duration  int64  `json:"-"`
	Steps     *int   `json:"steps,omitempty"`
	Status    string `json:"status"`
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
		return SetLogger(logp.L())
	}

	return eventLogger
}

func extractRunInfo(event *beat.Event) (*MonitorRunInfo, error) {
	errors := []error{}
	monitorID, err := event.GetValue("monitor.id")
	if err != nil {
		errors = append(errors, err)
	}

	durationUs, err := event.GetValue("monitor.duration.us")
	if err != nil {
		errors = append(errors, err)
	}

	monType, err := event.GetValue("monitor.type")
	if err != nil {
		errors = append(errors, err)
	}

	status, err := event.GetValue("monitor.status")
	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("logErrors: %+v", errors)
	}

	monitor := MonitorRunInfo{
		MonitorID: monitorID.(string),
		Type:      monType.(string),
		Duration:  durationUs.(int64),
		Status:    status.(string),
	}

	sc, _ := event.Meta.GetValue(META_STEP_COUNT)
	stepCount, ok := sc.(int)
	if ok {
		monitor.Steps = &stepCount
	}

	return &monitor, nil
}

func LogRun(event *beat.Event) {
	monitor, err := extractRunInfo(event)
	if err != nil {
		getLogger().Errorw("error gathering information to log event: ", err)
		return
	}

	getLogger().Infow(
		"Monitor finished",
		logp.Any("event", map[string]string{"action": ActionMonitorRun}),
		logp.Any("monitor", monitor),
	)
}
