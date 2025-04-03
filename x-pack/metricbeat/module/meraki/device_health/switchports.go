// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_health

import (
	"errors"
	"fmt"
	"time"

	meraki "github.com/meraki/dashboard-api-go/v3/sdk"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type switchport struct {
	port       *meraki.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts
	portStatus *meraki.ResponseItemSwitchGetDeviceSwitchPortsStatuses
}

func getDeviceSwitchports(client *meraki.Client, organizationID string, devices map[Serial]*Device, period time.Duration) error {
	switches, res, err := client.Switch.GetOrganizationSwitchPortsBySwitch(organizationID, &meraki.GetOrganizationSwitchPortsBySwitchQueryParams{})
	if err != nil {
		return fmt.Errorf("GetOrganizationSwitchPortsBySwitch failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
	}

	if switches == nil {
		return errors.New("GetOrganizationSwitchPortsBySwitch returned nil")
	}

	for _, device := range *switches {
		if device.Ports == nil {
			continue
		}

		var switchports []*switchport
		for i := range *device.Ports {
			switchports = append(switchports, &switchport{port: &(*device.Ports)[i]})
		}

		statuses, res, err := client.Switch.GetDeviceSwitchPortsStatuses(device.Serial, &meraki.GetDeviceSwitchPortsStatusesQueryParams{
			Timespan: period.Seconds(),
		})
		if err != nil {
			return fmt.Errorf("GetDeviceSwitchPortsStatuses failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
		}

		// match status to the port attributes found earlier using the shared port ID
		for i := range *statuses {
			status := (*statuses)[i]
			for _, switchport := range switchports {
				if switchport.port.PortID == status.PortID {
					switchport.portStatus = &status
					break
				}
			}
		}

		devices[Serial(device.Serial)].switchports = switchports
	}

	return nil
}

func reportSwitchportMetrics(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device) {
	metrics := []mapstr.M{}
	for _, device := range devices {
		if device == nil || device.details == nil {
			continue
		}
		for _, switchport := range device.switchports {
			if switchport == nil {
				continue
			}
			metric := deviceDetailsToMapstr(device.details)

			if switchport.port != nil {
				metric["switch.port.id"] = switchport.port.PortID
				metric["switch.port.access_policy_type"] = switchport.port.AccessPolicyType
				metric["switch.port.allowed_vlans"] = switchport.port.AllowedVLANs
				metric["switch.port.enabled"] = switchport.port.Enabled
				metric["switch.port.link_negotiation"] = switchport.port.LinkNegotiation
				metric["switch.port.name"] = switchport.port.Name
				metric["switch.port.poe_enabled"] = switchport.port.PoeEnabled
				metric["switch.port.rstp_enabled"] = switchport.port.RstpEnabled
				metric["switch.port.sticky_mac_allow_list"] = switchport.port.StickyMacAllowList
				metric["switch.port.sticky_mac_allow_list_limit"] = switchport.port.StickyMacAllowListLimit
				metric["switch.port.stp_guard"] = switchport.port.StpGuard
				metric["switch.port.tags"] = switchport.port.Tags
				metric["switch.port.type"] = switchport.port.Type
				metric["switch.port.vlan"] = switchport.port.VLAN
				metric["switch.port.voice_vlan"] = switchport.port.VoiceVLAN
			}

			if switchport.portStatus != nil {
				metric["switch.port.status.client_count"] = switchport.portStatus.ClientCount
				metric["switch.port.status.duplex"] = switchport.portStatus.Duplex
				metric["switch.port.status.enabled"] = switchport.portStatus.Enabled
				metric["switch.port.status.errors"] = switchport.portStatus.Errors
				metric["switch.port.status.is_uplink"] = switchport.portStatus.IsUplink
				metric["switch.port.status.power_usage_in_wh"] = switchport.portStatus.PowerUsageInWh
				metric["switch.port.status.speed"] = switchport.portStatus.Speed
				metric["switch.port.status.status"] = switchport.portStatus.Status
				metric["switch.port.status.warnings"] = switchport.portStatus.Warnings

				if switchport.portStatus.Cdp != nil {
					metric["switch.port.status.cdp.address"] = switchport.portStatus.Cdp.Address
					metric["switch.port.status.cdp.capabilities"] = switchport.portStatus.Cdp.Capabilities
					metric["switch.port.status.cdp.device_id"] = switchport.portStatus.Cdp.DeviceID
					metric["switch.port.status.cdp.management_address"] = switchport.portStatus.Cdp.ManagementAddress
					metric["switch.port.status.cdp.native_vlan"] = switchport.portStatus.Cdp.NativeVLAN
					metric["switch.port.status.cdp.platform"] = switchport.portStatus.Cdp.Platform
					metric["switch.port.status.cdp.port_id"] = switchport.portStatus.Cdp.PortID
					metric["switch.port.status.cdp.system_name"] = switchport.portStatus.Cdp.SystemName
					metric["switch.port.status.cdp.version"] = switchport.portStatus.Cdp.Version
					metric["switch.port.status.cdp.vtp_management_domain"] = switchport.portStatus.Cdp.VtpManagementDomain
				}

				if switchport.portStatus.Lldp != nil {
					metric["switch.port.status.lldp.chassis_id"] = switchport.portStatus.Lldp.ChassisID
					metric["switch.port.status.lldp.management_address"] = switchport.portStatus.Lldp.ManagementAddress
					metric["switch.port.status.lldp.management_vlan"] = switchport.portStatus.Lldp.ManagementVLAN
					metric["switch.port.status.lldp.port_description"] = switchport.portStatus.Lldp.PortDescription
					metric["switch.port.status.lldp.port_id"] = switchport.portStatus.Lldp.PortID
					metric["switch.port.status.lldp.port_vlan"] = switchport.portStatus.Lldp.PortVLAN
					metric["switch.port.status.lldp.system_capabilities"] = switchport.portStatus.Lldp.SystemCapabilities
					metric["switch.port.status.lldp.system_description"] = switchport.portStatus.Lldp.SystemDescription
					metric["switch.port.status.lldp.system_name"] = switchport.portStatus.Lldp.SystemName
				}

				if switchport.portStatus.SecurePort != nil {
					metric["switch.port.status.secure_port.active"] = switchport.portStatus.SecurePort.Active
					metric["switch.port.status.secure_port.authentication_status"] = switchport.portStatus.SecurePort.AuthenticationStatus
					metric["switch.port.status.secure_port.enabled"] = switchport.portStatus.SecurePort.Enabled

					if switchport.portStatus.SecurePort.ConfigOverrides != nil {
						metric["switch.port.status.secure_port.config_overrides.allowed_vlans"] = switchport.portStatus.SecurePort.ConfigOverrides.AllowedVLANs
						metric["switch.port.status.secure_port.config_overrides.type"] = switchport.portStatus.SecurePort.ConfigOverrides.Type
						metric["switch.port.status.secure_port.config_overrides.vlan"] = switchport.portStatus.SecurePort.ConfigOverrides.VLAN
						metric["switch.port.status.secure_port.config_overrides.voice_vlan"] = switchport.portStatus.SecurePort.ConfigOverrides.VoiceVLAN
					}
				}

				if switchport.portStatus.SpanningTree != nil {
					metric["switch.port.status.stp_statuses"] = switchport.portStatus.SpanningTree.Statuses
				}

				if switchport.portStatus.TrafficInKbps != nil {
					metric["switch.port.status.throughput.recv"] = switchport.portStatus.TrafficInKbps.Recv
					metric["switch.port.status.throughput.sent"] = switchport.portStatus.TrafficInKbps.Sent
					metric["switch.port.status.throughput.total"] = switchport.portStatus.TrafficInKbps.Total
				}

				if switchport.portStatus.UsageInKb != nil {
					metric["switch.port.status.usage.recv"] = switchport.portStatus.UsageInKb.Recv
					metric["switch.port.status.usage.sent"] = switchport.portStatus.UsageInKb.Sent
					metric["switch.port.status.usage.total"] = switchport.portStatus.UsageInKb.Total
				}
			}

			metrics = append(metrics, metric)
		}
	}

	reportMetricsForOrganization(reporter, organizationID, metrics)
}
