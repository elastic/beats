// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"

	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/management/status"
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
	var evt *componentstatus.Event
	switch s {
	case status.Starting:
		evt = componentstatus.NewEvent(componentstatus.StatusStarting)
	case status.Running:
		evt = componentstatus.NewEvent(componentstatus.StatusOK)
	case status.Degraded:
		evt = componentstatus.NewRecoverableErrorEvent(errors.New(msg))
	case status.Failed:
		evt = componentstatus.NewPermanentErrorEvent(errors.New(msg))
	case status.Stopping:
		evt = componentstatus.NewEvent(componentstatus.StatusStopped)
	case status.Stopped:
		evt = componentstatus.NewEvent(componentstatus.StatusStopped)
	default:
		return
	}

	componentstatus.ReportStatus(m.host, evt)
}
