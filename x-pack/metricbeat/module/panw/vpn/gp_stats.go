// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vpn

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/panw"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const gpStatsQuery = "<show><global-protect-gateway><statistics></statistics></global-protect-gateway></show>"

func getGlobalProtectStatsEvents(m *MetricSet) ([]mb.Event, error) {

	var response GPStatsResponse

	output, err := m.client.Op(gpStatsQuery, panw.Vsys, nil, nil)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, fmt.Errorf("error querying GlobalProtect statistics: %w", err)
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, fmt.Errorf("error unmarshaling GlobalProtect statistics response: %w", err)
	}

	events := formatGPStatsEvents(m, response)

	return events, nil

}

func formatGPStatsEvents(m *MetricSet, response GPStatsResponse) []mb.Event {

	if len(response.Result.Gateways) == 0 {
		return nil
	}

	events := make([]mb.Event, 0, len(response.Result.Gateways))
	timestamp := time.Now()

	totalCurrent := response.Result.TotalCurrentUsers
	totalPrevious := response.Result.TotalPreviousUsers

	for _, gateway := range response.Result.Gateways {
		event := mb.Event{
			Timestamp: timestamp,
			MetricSetFields: mapstr.M{
				"globalprotect.gateway.name":           gateway.Name,
				"globalprotect.gateway.current_users":  gateway.CurrentUsers,
				"globalprotect.gateway.previous_users": gateway.PreviousUsers,
				"globalprotect.total_current_users":    totalCurrent,
				"globalprotect.total_previous_users":   totalPrevious,
			},
			RootFields: mapstr.M{
				"observer.ip":     m.config.HostIp,
				"host.ip":         m.config.HostIp,
				"observer.vendor": "Palo Alto",
				"observer.type":   "firewall",
			}}

		events = append(events, event)
	}

	return events
}
