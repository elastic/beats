// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ha_interfaces

import (
	"encoding/xml"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/panos"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/PaloAltoNetworks/pango"
)

const (
	metricsetName = "ha_interfaces"
	vsys          = ""
	query         = "<show><high-availability><all></all></high-availability></show>"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(panos.ModuleName, metricsetName, New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	config panos.Config
	logger *logp.Logger
	client *pango.Firewall
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The panos ha_interfaces metricset is beta.")

	config := panos.Config{}
	logger := logp.NewLogger(base.FullyQualifiedName())

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	logger.Debugf("panos_ha_interfaces metricset config: %v", config)

	client := &pango.Firewall{Client: pango.Client{Hostname: config.HostIp, ApiKey: config.ApiKey}}

	return &MetricSet{
		BaseMetricSet: base,
		config:        config,
		logger:        logger,
		client:        client,
	}, nil
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	log := m.Logger()
	var response Response

	// Initialize the client
	if err := m.client.Initialize(); err != nil {
		log.Error("Failed to initialize client: %s", err)
		return err
	}
	log.Debug("panos_ha_interfaces.Fetch initialized client")

	output, err := m.client.Op(query, vsys, nil, nil)
	if err != nil {
		log.Error("Error: %s", err)
		return err
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		log.Error("Error: %s", err)
		return err
	}

	events := getEvents(m, response.Result)
	for _, event := range events {
		report.Event(event)
	}

	return nil
}

func getEvents(m *MetricSet, input Result) []mb.Event {
	events := make([]mb.Event, 0, len(input.Group.LinkMonitoring.Groups)+1)
	group := input.Group

	groupEvent := makeGroupEvent(m, input)
	events = append(events, *groupEvent)
	linkMonitorEvents := makeLinkMonitoringEvents(m, group.LinkMonitoring)
	events = append(events, linkMonitorEvents...)

	return events
}

func makeGroupEvent(m *MetricSet, input Result) *mb.Event {
	group := input.Group
	currentTime := time.Now()
	event := mb.Event{MetricSetFields: mapstr.M{
		"enabled":                               input.Enabled,
		"mode":                                  group.Mode,
		"running_sync":                          group.RunningSync,
		"running_sync_enabled":                  group.RunningSyncEnabled,
		"local_info.version":                    group.LocalInfo.Version,
		"local_info.state":                      group.LocalInfo.State,
		"local_info.state_duration":             group.LocalInfo.StateDuration,
		"local_info.mgmt_ip":                    group.LocalInfo.MgmtIP,
		"local_info.preemptive":                 group.LocalInfo.Preemptive,
		"local_info.mode":                       group.LocalInfo.Mode,
		"local_info.platform_model":             group.LocalInfo.PlatformModel,
		"local_info.state_sync":                 group.LocalInfo.StateSync,
		"local_info.state_sync_type":            group.LocalInfo.StateSyncType,
		"local_info.ha1_ipaddr":                 group.LocalInfo.HA1IPAddr,
		"local_info.ha1_macaddr":                group.LocalInfo.HA1MACAddr,
		"local_info.ha1_port":                   group.LocalInfo.HA1Port,
		"local_info.ha1_backup_ipaddr":          group.LocalInfo.HA1BackupIPAddr,
		"local_info.ha1_backup_macaddr":         group.LocalInfo.HA1BackupMACAddr,
		"local_info.ha1_backup_port":            group.LocalInfo.HA1BackupPort,
		"local_info.ha1_backup_gateway":         group.LocalInfo.HA1BackupGateway,
		"local_info.ha2_ipaddr":                 group.LocalInfo.HA2IPAddr,
		"local_info.ha2_macaddr":                group.LocalInfo.HA2MACAddr,
		"local_info.ha2_port":                   group.LocalInfo.HA2Port,
		"local_info.build_rel":                  group.LocalInfo.BuildRel,
		"local_info.url_version":                group.LocalInfo.URLVersion,
		"local_info.app_version":                group.LocalInfo.AppVersion,
		"local_info.iot_version":                group.LocalInfo.IoTVersion,
		"local_info.av_version":                 group.LocalInfo.AVVersion,
		"local_info.threat_version":             group.LocalInfo.ThreatVersion,
		"local_info.vpn_client_version":         group.LocalInfo.VPNClientVersion,
		"local_info.gp_client_version":          group.LocalInfo.GPClientVersion,
		"peer_info.conn_status":                 group.PeerInfo.ConnStatus,
		"peer_info.state":                       group.PeerInfo.State,
		"peer_info.state_duration":              group.PeerInfo.StateDuration,
		"peer_info.mgmt_ip":                     group.PeerInfo.MgmtIP,
		"peer_info.preemptive":                  group.PeerInfo.Preemptive,
		"peer_info.mode":                        group.PeerInfo.Mode,
		"peer_info.platform_model":              group.PeerInfo.PlatformModel,
		"peer_info.priority":                    group.PeerInfo.Priority,
		"peer_info.ha1_ipaddr":                  group.PeerInfo.HA1IPAddr,
		"peer_info.ha1_macaddr":                 group.PeerInfo.HA1MACAddr,
		"peer_info.ha1_backup_ipaddr":           group.PeerInfo.HA1BackupIPAddr,
		"peer_info.ha1_backup_macaddr":          group.PeerInfo.HA1BackupMACAddr,
		"peer_info.ha2_ipaddr":                  group.PeerInfo.HA2IPAddr,
		"peer_info.ha2_macaddr":                 group.PeerInfo.HA2MACAddr,
		"peer_info.conn_ha1.status":             group.PeerInfo.ConnHA1.Status,
		"peer_info.conn_ha1.primary":            group.PeerInfo.ConnHA1.Primary,
		"peer_info.conn_ha1.description":        group.PeerInfo.ConnHA1.Desc,
		"peer_info.conn_ha2.status":             group.PeerInfo.ConnHA2.Status,
		"peer_info.conn_ha2.primary":            group.PeerInfo.ConnHA2.Primary,
		"peer_info.conn_ha2.description":        group.PeerInfo.ConnHA2.Desc,
		"peer_info.conn_ha1_backup.status":      group.PeerInfo.ConnHA1Backup.Status,
		"peer_info.conn_ha1_backup.description": group.PeerInfo.ConnHA1Backup.Desc,
		"link_monitoring.enabled":               group.LinkMonitoring.Enabled,
	},
	}

	event.Timestamp = currentTime
	event.RootFields = mapstr.M{
		"observer.ip":     m.config.HostIp,
		"host.ip":         m.config.HostIp,
		"observer.vendor": "Palo Alto",
		"observer.type":   "firewall",
	}

	return &event
}

func makeLinkMonitoringEvents(m *MetricSet, links LinkMonitoring) []mb.Event {
	events := make([]mb.Event, 0, len(links.Groups))
	currentTime := time.Now()
	var event mb.Event
	for _, group := range links.Groups {
		for _, interface_entry := range group.Interface {
			event = mb.Event{MetricSetFields: mapstr.M{
				"link_monitoring.enabled":                 links.Enabled,
				"link_monitoring.failure_condition":       links.FailureCondition,
				"link_monitoring.group.name":              group.Name,
				"link_monitoring.group.enabled":           group.Enabled,
				"link_monitoring.group.failure_condition": group.FailureCondition,
				"link_monitoring.group.interface.name":    interface_entry.Name,
				"link_monitoring.group.interface.status":  interface_entry.Status,
			}}
		}

		event.Timestamp = currentTime
		event.RootFields = mapstr.M{
			"observer.ip": m.config.HostIp,
		}

		events = append(events, event)
	}

	return events
}
