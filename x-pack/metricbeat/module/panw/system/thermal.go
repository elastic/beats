// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"encoding/xml"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/panw"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const thermalQuery = "<show><system><environmentals><thermal></thermal></environmentals></system></show>"

func getThermalEvents(m *MetricSet) ([]mb.Event, error) {
	var response ThermalResponse

	output, err := m.client.Op(thermalQuery, panw.Vsys, nil, nil)
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
	if response == nil || len(response.Result.Thermal.Slots) == 0 {
		return nil
	}

	events := make([]mb.Event, 0, len(response.Result.Thermal.Slots))
	timestamp := time.Now()
	var event mb.Event

	for _, slot := range response.Result.Thermal.Slots {
		for _, entry := range slot.Entries {
			alarm, err := panw.StringToBool(entry.Alarm)
			if err != nil {
				m.logger.Warn("Failed to convert alarm value %s to boolean: %s. Defaulting to false.", entry.Alarm, err)
			}
			m.logger.Debugf("Processing slot %d entry %+v", entry.Slot, entry)
			event = mb.Event{
				Timestamp: timestamp,
				MetricSetFields: mapstr.M{
					"thermal.slot_number":     entry.Slot,
					"thermal.description":     entry.Description,
					"thermal.alarm":           alarm,
					"thermal.degress_celsius": entry.DegreesCelsius,
					"thermal.minimum_temp":    entry.MinimumTemp,
					"thermal.maximum_temp":    entry.MaximumTemp,
				},
				RootFields: mapstr.M{
					"observer.ip":     m.config.HostIp,
					"host.ip":         m.config.HostIp,
					"observer.vendor": "Palo Alto",
					"observer.type":   "firewall",
				}}
		}

		events = append(events, event)
	}

	return events
}
