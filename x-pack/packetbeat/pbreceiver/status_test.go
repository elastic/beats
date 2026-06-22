// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pbreceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

// TestReceiverStatus verifies that the packetbeat receiver propagates
// component status events to the OTel host when running under the OTel
// collector. It checks that a StatusOK event is reported at least once
// while the receiver processes a pre-recorded pcap file — no live
// packet-capture capability (root / npcap) required.
func TestReceiverStatus(t *testing.T) {
	inputID := "pb-status-test"

	config := Config{
		Beatconfig: map[string]any{
			"packetbeat": map[string]any{
				// "id" is read by otelstatus.getInputId from pb.config
				// (the packetbeat sub-section), producing the sub-reporter key.
				"id": inputID,
				"interfaces": map[string]any{
					// Read from a pre-recorded pcap file; no libpcap / root required.
					"file": "../../../packetbeat/tests/system/pcaps/http_x_forwarded_for.pcap",
				},
				"protocols": []map[string]any{
					{
						"type":  "http",
						"ports": []int{80},
					},
				},
			},
			"logging": map[string]any{
				"level":     "info",
				"selectors": []string{"*"},
			},
			"path.home":               t.TempDir(),
			"management.otel.enabled": true,
		},
	}

	factory := NewFactoryWithSettings(Settings{Home: t.TempDir()})
	set := receiver.Settings{
		ID: component.NewIDWithName(factory.Type(), "r1"),
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
	}

	host := &oteltest.MockHost{}
	rec, err := factory.CreateLogs(t.Context(), set, &config, consumertest.NewNop())
	require.NoError(t, err)
	require.NoError(t, rec.Start(t.Context(), host))

	// Wait for StatusOK to be reported. The sniffer starts, emits Running
	// (→ StatusOK), reads the pcap, then the sub-runner records Stopped —
	// all within milliseconds. The aggregate reporter ignores Stopped and
	// keeps reporting StatusOK (Running), so we poll the full event history
	// to catch the first StatusOK even if the pcap finished before the first
	// poll tick.
	// Scan all events looking for one where both the top-level component status
	// and the sub-runner's reported status are StatusOK. The aggregate reporter
	// always emits StatusOK (it treats Stopped sub-runners as Running), so we
	// must also check the inputs map to confirm the receiver was actually running,
	// not just stopped.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		evts := host.GetEvents()
		var found bool
		for _, evt := range evts {
			if evt.Status() != componentstatus.StatusOK {
				continue
			}
			attrs := evt.Attributes().AsRaw()
			inputs, ok := attrs["inputs"].(map[string]any)
			if !ok {
				continue
			}
			entry, ok := inputs[inputID].(map[string]any)
			if !ok {
				continue
			}
			if entry["status"] == componentstatus.StatusOK.String() {
				found = true
				break
			}
		}
		assert.True(c, found,
			"no event found where both component status and input %q status are StatusOK; all events: %v",
			inputID, evts)
	}, 30*time.Second, 10*time.Millisecond,
		"timeout waiting for StatusOK from packetbeat receiver")

	require.NoError(t, rec.Shutdown(t.Context()))
}
