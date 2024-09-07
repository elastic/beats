// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"encoding/xml"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func getPowerEvents(m *MetricSet) ([]mb.Event, error) {

	query := "<show><system><environmentals><power></power></environmentals></system></show>"
	var response PowerResponse

	output, err := m.client.Op(query, vsys, nil, nil)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, err
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, err
	}

	events := formatPowerEvents(m, &response)

	return events, nil
}

func formatPowerEvents(m *MetricSet, response *PowerResponse) []mb.Event {
	log := m.Logger()
	events := make([]mb.Event, 0, len(response.Result.Power.Slots))
	currentTime := time.Now()
	var event mb.Event
	for _, slot := range response.Result.Power.Slots {
		for _, entry := range slot.Entries {
			log.Debugf("Processing slot %d entry %+v", entry.Slot, entry)
			event = mb.Event{MetricSetFields: mapstr.M{

				"slot_number":   entry.Slot,
				"description":   entry.Description,
				"alarm":         entry.Alarm,
				"volts":         entry.Volts,
				"minimum_volts": entry.MinimumVolts,
				"maximum_volts": entry.MaximumVolts,
			},
				RootFields: mapstr.M{
					"observer.ip":     m.config.HostIp,
					"host.ip":         m.config.HostIp,
					"observer.vendor": "Palo Alto",
					"observer.type":   "firewall",
					"@Timestamp":      currentTime,
				}}
		}

		events = append(events, event)
	}

	return events
}
