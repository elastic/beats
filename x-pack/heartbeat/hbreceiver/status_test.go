// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"testing"

	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

func TestReceiverStatus(t *testing.T) {
	// Use a TCP monitor with a long schedule so the first check never fires
	// during the test. Monitor.Start() still calls updateStatus(Running)
	// immediately, producing the StatusOK event we assert below.
	monitorID := "test-tcp"
	inputStatusAttributes := func(state string, msg string) pcommon.Map {
		eventAttributes := pcommon.NewMap()
		inputStatuses := eventAttributes.PutEmptyMap("inputs")
		monitorStatus := inputStatuses.PutEmptyMap(monitorID)
		monitorStatus.PutStr("status", state)
		monitorStatus.PutStr("error", msg)
		return eventAttributes
	}

	testCases := []struct {
		name   string
		status *componentstatus.Event
	}{
		{
			name: "running monitor",
			status: componentstatus.NewEvent(componentstatus.StatusOK,
				componentstatus.WithAttributes(inputStatusAttributes(componentstatus.StatusOK.String(), ""))),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			config := Config{
				Beatconfig: map[string]any{
					"heartbeat": map[string]any{
						"monitors": []map[string]any{{
							"type": "tcp", "id": monitorID,
							"schedule": "@every 30s", "timeout": "3s",
							"hosts": []string{"localhost:0"},
						}},
					},
					"queue.mem.flush.timeout": "0s",
					"path.home":               t.TempDir(),
				},
			}
			oteltest.CheckReceivers(oteltest.CheckReceiversParams{
				T: t,
				Receivers: []oteltest.ReceiverConfig{
					{
						Name:    "r1",
						Beat:    "heartbeat",
						Config:  &config,
						Factory: NewFactoryWithSettings(Settings{Home: t.TempDir()}),
					},
				},
				Status: test.status,
			})
		})
	}
}
