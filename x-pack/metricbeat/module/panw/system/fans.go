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

const fansQuery = "<show><system><environmentals><fans></fans></environmentals></system></show>"

func getFanEvents(m *MetricSet) ([]mb.Event, error) {

	var response FanResponse

	output, err := m.client.Op(fansQuery, panw.Vsys, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error querying fan data: %w", err)
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling fan data: %w", err)
	}

	return formatFanEvents(m, &response), nil
}

func formatFanEvents(m *MetricSet, response *FanResponse) []mb.Event {
	if response == nil || len(response.Result.Fan.Slots) == 0 {
		return nil
	}

	events := make([]mb.Event, 0)
	timestamp := time.Now()

	for _, slot := range response.Result.Fan.Slots {
		for _, entry := range slot.Entries {
			alarm, err := panw.StringToBool(entry.Alarm)
			if err != nil {
				m.logger.Warn("Failed to convert alarm value %s to boolean: %s. Defaulting to false.", entry.Alarm, err)
			}
			m.Logger().Debugf("Processing slot %d entry %+v", entry.Slot, entry)
			event := mb.Event{
				Timestamp: timestamp,
				MetricSetFields: mapstr.M{
					"fan.slot_number": entry.Slot,
					"fan.description": entry.Description,
					"fan.alarm":       alarm,
					"fan.rpm":         entry.RPMs,
					"fan.min_rpm":     entry.Min,
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
