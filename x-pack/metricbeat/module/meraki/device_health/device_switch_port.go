package device_health

import (
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"

	meraki_api "github.com/tommyers-elastic/dashboard-api-go/v3/sdk"
)

func reportOrganizationDeviceSwitchPortBySwitch(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device, orgSwitchPortsBySwitch *meraki_api.ResponseSwitchGetOrganizationSwitchPortsBySwitch) {

	metrics := []mapstr.M{}

	for _, switchPort := range *orgSwitchPortsBySwitch {

		port_encountered := false

		metric := mapstr.M{
			"switch.port.name":         switchPort.Name,
			"switch.port.serial":       switchPort.Serial,
			"switch.port.mac":          switchPort.Mac,
			"switch.port.network.name": switchPort.Network.Name,
			"switch.port.network.id":   switchPort.Network.ID,
			"switch.port.model":        switchPort.Model,
		}

		if device, ok := devices[Serial(switchPort.Serial)]; ok {
			metric["device.address"] = device.Address
			metric["device.firmware"] = device.Firmware
			metric["device.imei"] = device.Imei
			metric["device.lan_ip"] = device.LanIP
			metric["device.location"] = device.Location
			metric["device.mac"] = device.Mac
			metric["device.model"] = device.Model
			metric["device.name"] = device.Name
			metric["device.network_id"] = device.NetworkID
			metric["device.notes"] = device.Notes
			metric["device.product_type"] = device.ProductType
			metric["device.serial"] = device.Serial
			metric["device.tags"] = device.Tags

		}

		for _, port := range *switchPort.Ports {
			metric["switch.port.port_id"] = port.PortID
			metric["switch.port.name"] = port.Name
			metric["switch.port.tags"] = port.Tags
			metric["switch.port.enabled"] = port.Enabled
			metric["switch.port.poe_enabled"] = port.PoeEnabled
			metric["switch.port.vlan"] = port.VLAN
			metric["switch.port.voice_vlan"] = port.VoiceVLAN
			metric["switch.port.allowed_vlans"] = port.AllowedVLANs
			metric["switch.port.rstp_enabled"] = port.RstpEnabled
			metric["switch.port.stp_guard"] = port.StpGuard
			metric["switch.port.link_negotiation"] = port.LinkNegotiation
			metric["switch.port.access_policy_type"] = port.AccessPolicyType
			metric["switch.port.sticky_mac_allow_list"] = port.StickyMacAllowList
			metric["switch.port.sticky_mac_allow_list_limit"] = port.StickyMacAllowListLimit
			port_encountered = true
			metrics = append(metrics, metric)
		}

		if !port_encountered {
			metrics = append(metrics, metric)
		}

	}

	ReportMetricsForOrganization(reporter, organizationID, metrics)
}
