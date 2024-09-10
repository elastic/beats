// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/panw"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const powerQuery = "<show><system><environmentals><power></power></environmentals></system></show>"

// getPowerEvents retrieves power-related events from a PAN-OS device.
func getPowerEvents(m *MetricSet) ([]mb.Event, error) {

	var response PowerResponse

	output, err := m.client.Op(powerQuery, panw.Vsys, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute operation: %w", err)
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML response: %w", err)
	}

	if len(response.Result.Power.Slots) == 0 {
		m.logger.Warn("No power events found in the response")
		return nil, nil
	}

	events := formatPowerEvents(m, &response)

	return events, nil
}

func formatPowerEvents(m *MetricSet, response *PowerResponse) []mb.Event {
	events := make([]mb.Event, 0)
	timestamp := time.Now()

	for _, slot := range response.Result.Power.Slots {
		for _, entry := range slot.Entries {
			m.Logger().Debugf("Processing slot %d entry %+v", entry.Slot, entry)
			event := mb.Event{
				Timestamp: timestamp,
				MetricSetFields: mapstr.M{
					"power.slot_number":   entry.Slot,
					"power.description":   entry.Description,
					"power.alarm":         entry.Alarm,
					"power.volts":         entry.Volts,
					"power.minimum_volts": entry.MinimumVolts,
					"power.maximum_volts": entry.MaximumVolts,
				},
				RootFields: mapstr.M{
					"observer.ip":     m.config.HostIp,
					"host.ip":         m.config.HostIp,
					"observer.vendor": "Palo Alto",
					"observer.type":   "firewall",
				},
			}
			events = append(events, event)
		}
	}

	return events
}
