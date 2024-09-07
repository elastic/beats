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
			"globalprotect.domain":                 session.Domain,
			"globalprotect.is_local":               session.IsLocal,
			"globalprotect.username":               session.Username,
			"globalprotect.rimary_username":        session.PrimaryUsername,
			"globalprotect.region_for_config":      session.RegionForConfig,
			"globalprotect.ource_region":           session.SourceRegion,
			"globalprotect.computer":               session.Computer,
			"globalprotect.client":                 session.Client,
			"globalprotect.vpn_type":               session.VPNType,
			"globalprotect.host_id":                session.HostID,
			"globalprotect.app_version":            session.AppVersion,
			"globalprotect.virtual_ip":             session.VirtualIP,
			"globalprotect.virtual_ipv6":           session.VirtualIPv6,
			"globalprotect.public_ip":              session.PublicIP,
			"globalprotect.public_ipv6":            session.PublicIPv6,
			"globalprotect.tunnel_type":            session.TunnelType,
			"globalprotect.public_connection_ipv6": session.PublicConnectionIPv6,
			"globalprotect.lient_ip":               session.ClientIP,
			"globalprotect.login_time":             session.LoginTime,
			"globalprotect.login_time_utc":         session.LoginTimeUTC,
			"globalprotect.lifetime":               session.Lifetime,
			"globalprotect.request_login":          session.RequestLogin,
			"globalprotect.request_get_config":     session.RequestGetConfig,
			"globalprotect.request_sslvpn_connect": session.RequestSSLVPNConnect,
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
