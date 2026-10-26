// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap/zaptest"

	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

// collectorHost records the status the receiver reports, and like the
// collector, rejects status changes out of StatusPermanentError other than
// stopping.
type collectorHost struct {
	oteltest.MockHost

	mx       sync.Mutex
	current  componentstatus.Status
	rejected []string
}

func (h *collectorHost) Report(evt *componentstatus.Event) {
	h.mx.Lock()
	if h.current == componentstatus.StatusPermanentError && evt.Status() != componentstatus.StatusStopping {
		h.rejected = append(h.rejected, h.current.String()+" -> "+evt.Status().String())
		h.mx.Unlock()
		return
	}
	h.current = evt.Status()
	h.mx.Unlock()
	h.MockHost.Report(evt)
}

func (h *collectorHost) rejectedChanges() []string {
	h.mx.Lock()
	defer h.mx.Unlock()
	return append([]string(nil), h.rejected...)
}

// TestReceiverRecoversFromFailedMonitor checks that the receiver reports a
// monitor that can't be created as a failure it can recover from: once the
// monitor is fixed, the receiver reports it running, which the collector
// would refuse after a permanent error.
func TestReceiverRecoversFromFailedMonitor(t *testing.T) {
	configDir := t.TempDir()
	factory := NewFactoryWithSettings(Settings{Home: t.TempDir()})
	set := receiver.Settings{}
	set.ID = component.NewIDWithName(factory.Type(), "recover")
	set.Logger = zaptest.NewLogger(t)
	rec, err := factory.CreateLogs(t.Context(), set, dynamicMonitorConfig(t, configDir), consumertest.NewNop())
	require.NoError(t, err)
	host := &collectorHost{}
	require.NoError(t, rec.Start(t.Context(), host))
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	monitor := func(schedule string) string {
		return `
- type: tcp
  id: monitor
  schedule: "` + schedule + `"
  hosts: ["localhost:0"]
`
	}
	inputStatus := func(c *assert.CollectT) (componentstatus.Status, string) {
		evt := host.GetEvent()
		if !assert.NotNil(c, evt) {
			return componentstatus.StatusNone, ""
		}
		inputs, ok := evt.Attributes().Get(inputStatusesKey)
		if !assert.True(c, ok, "no input statuses") {
			return evt.Status(), ""
		}
		input, ok := inputs.Map().Get("monitor")
		if !assert.True(c, ok, "no status for the monitor") {
			return evt.Status(), ""
		}
		status, _ := input.Map().Get("status")
		return evt.Status(), status.Str()
	}

	// An invalid monitor is reported as failed, and the receiver as recoverable.
	writeMonitorYAML(t, configDir, "monitor.yml", monitor("not a schedule"))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		receiverStatus, monitorStatus := inputStatus(c)
		assert.Equal(c, componentstatus.StatusRecoverableError, receiverStatus)
		assert.Equal(c, componentstatus.StatusPermanentError.String(), monitorStatus)
	}, 30*time.Second, 100*time.Millisecond)

	// Fixing the monitor clears the failure.
	writeMonitorYAML(t, configDir, "monitor.yml", monitor("@every 30s"))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		receiverStatus, monitorStatus := inputStatus(c)
		assert.Equal(c, componentstatus.StatusOK, receiverStatus)
		assert.Equal(c, componentstatus.StatusOK.String(), monitorStatus)
	}, 30*time.Second, 100*time.Millisecond)
	assert.Empty(t, host.rejectedChanges(), "the collector would reject these status changes")
}
