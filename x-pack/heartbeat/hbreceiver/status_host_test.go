// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	otelstatus "github.com/elastic/beats/v7/x-pack/otel/status"
)

func TestStatusHost(t *testing.T) {
	inputs := pcommon.NewMap()
	monitor := inputs.PutEmptyMap(otelstatus.InputStatusAttributesKey).PutEmptyMap("monitor")
	monitor.PutStr("status", componentstatus.StatusPermanentError.String())
	monitor.PutStr("error", "invalid schedule")

	tests := map[string]struct {
		event *componentstatus.Event
		want  componentstatus.Status
	}{
		"failed monitor is recoverable": {
			event: componentstatus.NewEvent(componentstatus.StatusPermanentError,
				componentstatus.WithError(errors.New("invalid schedule")), componentstatus.WithAttributes(inputs)),
			want: componentstatus.StatusRecoverableError,
		},
		"failed receiver stays permanent": {
			event: componentstatus.NewEvent(componentstatus.StatusPermanentError,
				componentstatus.WithError(errors.New("heartbeat failed to start"))),
			want: componentstatus.StatusPermanentError,
		},
		"other statuses are unchanged": {
			event: componentstatus.NewEvent(componentstatus.StatusOK, componentstatus.WithAttributes(inputs)),
			want:  componentstatus.StatusOK,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			host := &oteltest.MockHost{}
			statusHost{Host: host}.Report(tc.event)

			got := host.GetEvent()
			require.NotNil(t, got)
			assert.Equal(t, tc.want, got.Status())
			assert.Equal(t, tc.event.Err(), got.Err())
			assert.Equal(t, tc.event.Attributes().AsRaw(), got.Attributes().AsRaw(), "input statuses are unchanged")
		})
	}
}
