// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vpn

import (
	"encoding/xml"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func getGlobalProtectSessionEvents(m *MetricSet) ([]mb.Event, error) {
	query := "<show><global-protect-gateway><current-user></current-user></global-protect-gateway></show>"
	var response GPSessionsResponse

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

	events := formatGPSessionEvents(m, response.Result.Sessions)

	return events, nil

}

func formatGPSessionEvents(m *MetricSet, sessions []GPSession) []mb.Event {
	events := make([]mb.Event, 0, len(sessions))

	currentTime := time.Now()

	for _, session := range sessions {
		event := mb.Event{MetricSetFields: mapstr.M{
			"domain":                 session.Domain,
			"is_local":               session.IsLocal,
			"username":               session.Username,
			"primary_username":       session.PrimaryUsername,
			"region_for_config":      session.RegionForConfig,
			"source_region":          session.SourceRegion,
			"computer":               session.Computer,
			"client":                 session.Client,
			"vpn_type":               session.VPNType,
			"host_id":                session.HostID,
			"app_version":            session.AppVersion,
			"virtual_ip":             session.VirtualIP,
			"virtual_ipv6":           session.VirtualIPv6,
			"public_ip":              session.PublicIP,
			"public_ipv6":            session.PublicIPv6,
			"tunnel_type":            session.TunnelType,
			"public_connection_ipv6": session.PublicConnectionIPv6,
			"client_ip":              session.ClientIP,
			"login_time":             session.LoginTime,
			"login_time_utc":         session.LoginTimeUTC,
			"lifetime":               session.Lifetime,
			"request_login":          session.RequestLogin,
			"request_get_config":     session.RequestGetConfig,
			"request_sslvpn_connect": session.RequestSSLVPNConnect,
		},
			RootFields: mapstr.M{
				"observer.ip":     m.config.HostIp,
				"host.ip":         m.config.HostIp,
				"observer.vendor": "Palo Alto",
				"observer.type":   "firewall",
				"@Timestamp":      currentTime,
			}}

		events = append(events, event)
	}

	return events
}
