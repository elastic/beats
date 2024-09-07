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

func getThermalEvents(m *MetricSet) ([]mb.Event, error) {
	var response ThermalResponse

	query := "<show><system><environmentals><thermal></thermal></environmentals></system></show>"

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

	events := formatThermalEvents(m, &response)

	return events, nil

}

func formatThermalEvents(m *MetricSet, response *ThermalResponse) []mb.Event {
	log := m.Logger()
	events := make([]mb.Event, 0, len(response.Result.Thermal.Slots))
	currentTime := time.Now()
	var event mb.Event
	for _, slot := range response.Result.Thermal.Slots {
		for _, entry := range slot.Entries {
			log.Debugf("Processing slot %d entry %+v", entry.Slot, entry)
			event = mb.Event{MetricSetFields: mapstr.M{

				"thermal.slot_number":     entry.Slot,
				"thermal.description":     entry.Description,
				"thermal.alarm":           entry.Alarm,
				"thermal.degress_celsius": entry.DegreesCelsius,
				"thermal.minimum_temp":    entry.MinimumTemp,
				"thermal.maximum_temp":    entry.MaximumTemp,
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
