// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
)

// inputStatusesKey is the event attribute where the beat receiver status
// reporter puts the status of each input.
const inputStatusesKey = "inputs"

// statusHost adapts the status the receiver reports to the collector.
//
// The collector never lets a component leave StatusPermanentError, because a
// component only recovers from it by being recreated with a new config. The
// Heartbeat receiver also loads monitor configs outside its component config,
// from heartbeat.config.monitors, so a failed monitor can be fixed or removed
// while the receiver keeps running. A failed monitor is therefore reported as
// a recoverable error of the receiver; the monitor's own status in the event
// still shows that it failed. A failure of the receiver itself, reported
// without input statuses, stays permanent.
type statusHost struct {
	component.Host
}

func (h statusHost) Report(evt *componentstatus.Event) {
	if evt.Status() == componentstatus.StatusPermanentError {
		if _, fromInputs := evt.Attributes().Get(inputStatusesKey); fromInputs {
			evt = componentstatus.NewEvent(componentstatus.StatusRecoverableError,
				componentstatus.WithError(evt.Err()),
				componentstatus.WithAttributes(evt.Attributes()))
		}
	}
	componentstatus.ReportStatus(h.Host, evt)
}
