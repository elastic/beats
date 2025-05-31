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

package manager

import (
	"errors"

	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
)

type otelManager struct {
	management.Manager
	host component.Host
}

var _ management.Manager = (*otelManager)(nil)
var _ status.StatusReporter = (*otelManager)(nil)

func NewOtelManager(parent management.Manager, host component.Host) management.Manager {
	return &otelManager{
		Manager: parent,
		host:    host,
	}
}

func (m *otelManager) UpdateStatus(s status.Status, msg string) {
	switch s {
	case status.Starting:
		componentstatus.ReportStatus(m.host, componentstatus.NewEvent(componentstatus.StatusStarting))
	case status.Running:
		componentstatus.ReportStatus(m.host, componentstatus.NewEvent(componentstatus.StatusOK))
	case status.Degraded:
		componentstatus.ReportStatus(m.host, componentstatus.NewRecoverableErrorEvent(errors.New(msg)))
	case status.Failed:
		componentstatus.ReportStatus(m.host, componentstatus.NewPermanentErrorEvent(errors.New(msg)))
	case status.Stopping:
		componentstatus.ReportStatus(m.host, componentstatus.NewEvent(componentstatus.StatusStopped))
	case status.Stopped:
		componentstatus.ReportStatus(m.host, componentstatus.NewEvent(componentstatus.StatusStopped))
	}
}
