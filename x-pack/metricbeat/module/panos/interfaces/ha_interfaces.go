// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package interfaces

import (
	"encoding/xml"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func getHAInterfaceEvents(m *MetricSet) ([]mb.Event, error) {
	query := "<show><high-availability><all></all></high-availability></show>"
	var response HAResponse

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

	events := formatHAInterfaceEvents(m, response.Result)

	return events, nil

}

func formatHAInterfaceEvents(m *MetricSet, input HAResult) []mb.Event {
	events := make([]mb.Event, 0, len(input.Group.LinkMonitoring.Groups)+1)
	group := input.Group

	groupEvent := makeGroupEvent(m, input)
	events = append(events, *groupEvent)
	linkMonitorEvents := makeLinkMonitoringEvents(m, group.LinkMonitoring)
	events = append(events, linkMonitorEvents...)

	return events
}

func makeGroupEvent(m *MetricSet, input HAResult) *mb.Event {
	group := input.Group
	currentTime := time.Now()
	event := mb.Event{MetricSetFields: mapstr.M{
		"ha.enabled":                               input.Enabled,
		"ha.mode":                                  group.Mode,
		"ha.running_sync":                          group.RunningSync,
		"ha.running_sync_enabled":                  group.RunningSyncEnabled,
		"ha.local_info.version":                    group.LocalInfo.Version,
		"ha.local_info.state":                      group.LocalInfo.State,
		"ha.local_info.state_duration":             group.LocalInfo.StateDuration,
		"ha.local_info.mgmt_ip":                    group.LocalInfo.MgmtIP,
		"ha.local_info.preemptive":                 group.LocalInfo.Preemptive,
		"ha.local_info.mode":                       group.LocalInfo.Mode,
		"ha.local_info.platform_model":             group.LocalInfo.PlatformModel,
		"ha.local_info.state_sync":                 group.LocalInfo.StateSync,
		"ha.local_info.state_sync_type":            group.LocalInfo.StateSyncType,
		"ha.local_info.ha1_ipaddr":                 group.LocalInfo.HA1IPAddr,
		"ha.local_info.ha1_macaddr":                group.LocalInfo.HA1MACAddr,
		"ha.local_info.ha1_port":                   group.LocalInfo.HA1Port,
		"ha.local_info.ha1_backup_ipaddr":          group.LocalInfo.HA1BackupIPAddr,
		"ha.local_info.ha1_backup_macaddr":         group.LocalInfo.HA1BackupMACAddr,
		"ha.local_info.ha1_backup_port":            group.LocalInfo.HA1BackupPort,
		"ha.local_info.ha1_backup_gateway":         group.LocalInfo.HA1BackupGateway,
		"ha.local_info.ha2_ipaddr":                 group.LocalInfo.HA2IPAddr,
		"ha.local_info.ha2_macaddr":                group.LocalInfo.HA2MACAddr,
		"ha.local_info.ha2_port":                   group.LocalInfo.HA2Port,
		"ha.local_info.build_rel":                  group.LocalInfo.BuildRel,
		"ha.local_info.url_version":                group.LocalInfo.URLVersion,
		"ha.local_info.app_version":                group.LocalInfo.AppVersion,
		"ha.local_info.iot_version":                group.LocalInfo.IoTVersion,
		"ha.local_info.av_version":                 group.LocalInfo.AVVersion,
		"ha.local_info.threat_version":             group.LocalInfo.ThreatVersion,
		"ha.local_info.vpn_client_version":         group.LocalInfo.VPNClientVersion,
		"ha.local_info.gp_client_version":          group.LocalInfo.GPClientVersion,
		"ha.peer_info.conn_status":                 group.PeerInfo.ConnStatus,
		"ha.peer_info.state":                       group.PeerInfo.State,
		"ha.peer_info.state_duration":              group.PeerInfo.StateDuration,
		"ha.peer_info.mgmt_ip":                     group.PeerInfo.MgmtIP,
		"ha.peer_info.preemptive":                  group.PeerInfo.Preemptive,
		"ha.peer_info.mode":                        group.PeerInfo.Mode,
		"ha.peer_info.platform_model":              group.PeerInfo.PlatformModel,
		"ha.peer_info.priority":                    group.PeerInfo.Priority,
		"ha.peer_info.ha1_ipaddr":                  group.PeerInfo.HA1IPAddr,
		"ha.peer_info.ha1_macaddr":                 group.PeerInfo.HA1MACAddr,
		"ha.peer_info.ha1_backup_ipaddr":           group.PeerInfo.HA1BackupIPAddr,
		"ha.peer_info.ha1_backup_macaddr":          group.PeerInfo.HA1BackupMACAddr,
		"ha.peer_info.ha2_ipaddr":                  group.PeerInfo.HA2IPAddr,
		"ha.peer_info.ha2_macaddr":                 group.PeerInfo.HA2MACAddr,
		"ha.peer_info.conn_ha1.status":             group.PeerInfo.ConnHA1.Status,
		"ha.peer_info.conn_ha1.primary":            group.PeerInfo.ConnHA1.Primary,
		"ha.peer_info.conn_ha1.description":        group.PeerInfo.ConnHA1.Desc,
		"ha.peer_info.conn_ha2.status":             group.PeerInfo.ConnHA2.Status,
		"ha.peer_info.conn_ha2.primary":            group.PeerInfo.ConnHA2.Primary,
		"ha.peer_info.conn_ha2.description":        group.PeerInfo.ConnHA2.Desc,
		"ha.peer_info.conn_ha1_backup.status":      group.PeerInfo.ConnHA1Backup.Status,
		"ha.peer_info.conn_ha1_backup.description": group.PeerInfo.ConnHA1Backup.Desc,
		"ha.link_monitoring.enabled":               group.LinkMonitoring.Enabled,
	},
		RootFields: mapstr.M{
			"observer.ip":     m.config.HostIp,
			"host.ip":         m.config.HostIp,
			"observer.vendor": "Palo Alto",
			"observer.type":   "firewall",
			"@Timestamp":      currentTime,
		},
	}

	return &event
}

func makeLinkMonitoringEvents(m *MetricSet, links HALinkMonitoring) []mb.Event {
	events := make([]mb.Event, 0, len(links.Groups))
	currentTime := time.Now()
	var event mb.Event
	for _, group := range links.Groups {
		for _, interface_entry := range group.Interface {
			event = mb.Event{MetricSetFields: mapstr.M{
				"ha.link_monitoring.enabled":                  links.Enabled,
				"ha.llink_monitoring.failure_condition":       links.FailureCondition,
				"ha.llink_monitoring.group.name":              group.Name,
				"ha.llink_monitoring.group.enabled":           group.Enabled,
				"ha.llink_monitoring.group.failure_condition": group.FailureCondition,
				"ha.llink_monitoring.group.interface.name":    interface_entry.Name,
				"ha.llink_monitoring.group.interface.status":  interface_entry.Status,
			},
				RootFields: mapstr.M{
					"observer.ip":     m.config.HostIp,
					"host.ip":         m.config.HostIp,
					"observer.vendor": "Palo Alto",
					"observer.type":   "firewall",
					"@Timestamp":      currentTime,
				},
			}
		}

		events = append(events, event)
	}

	return events
}
