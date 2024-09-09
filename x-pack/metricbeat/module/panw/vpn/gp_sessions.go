// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vpn

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const gpSessionsQuery = "<show><global-protect-gateway><current-user></current-user></global-protect-gateway></show>"

func getGlobalProtectSessionEvents(m *MetricSet) ([]mb.Event, error) {
	var response GPSessionsResponse

	output, err := m.client.Op(gpSessionsQuery, vsys, nil, nil)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, fmt.Errorf("error querying GlobalProtect sessions: %w", err)
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, fmt.Errorf("error unmarshaling GlobalProtect sessions response: %w", err)
	}

	events := formatGPSessionEvents(m, response.Result.Sessions)

	return events, nil

}

func formatGPSessionEvents(m *MetricSet, sessions []GPSession) []mb.Event {
	if sessions == nil {
		return nil
	}

	events := make([]mb.Event, 0, len(sessions))
	timestamp := time.Now()

	for _, session := range sessions {
		event := mb.Event{
			Timestamp: timestamp,
			MetricSetFields: mapstr.M{
				"globalprotect.session.domain":                 session.Domain,
				"globalprotect.session.is_local":               session.IsLocal,
				"globalprotect.session.username":               session.Username,
				"globalprotect.session.rimary_username":        session.PrimaryUsername,
				"globalprotect.session.region_for_config":      session.RegionForConfig,
				"globalprotect.session.source_region":          session.SourceRegion,
				"globalprotect.session.computer":               session.Computer,
				"globalprotect.session.client":                 session.Client,
				"globalprotect.session.vpn_type":               session.VPNType,
				"globalprotect.session.host_id":                session.HostID,
				"globalprotect.session.app_version":            session.AppVersion,
				"globalprotect.session.virtual_ip":             session.VirtualIP,
				"globalprotect.session.virtual_ipv6":           session.VirtualIPv6,
				"globalprotect.session.public_ip":              session.PublicIP,
				"globalprotect.session.public_ipv6":            session.PublicIPv6,
				"globalprotect.session.tunnel_type":            session.TunnelType,
				"globalprotect.session.public_connection_ipv6": session.PublicConnectionIPv6,
				"globalprotect.session.lient_ip":               session.ClientIP,
				"globalprotect.session.login_time":             session.LoginTime,
				"globalprotect.session.login_time_utc":         session.LoginTimeUTC,
				"globalprotect.session.lifetime":               session.Lifetime,
				"globalprotect.session.request_login":          session.RequestLogin,
				"globalprotect.session.request_get_config":     session.RequestGetConfig,
				"globalprotect.session.request_sslvpn_connect": session.RequestSSLVPNConnect,
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
