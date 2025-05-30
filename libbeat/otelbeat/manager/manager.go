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
