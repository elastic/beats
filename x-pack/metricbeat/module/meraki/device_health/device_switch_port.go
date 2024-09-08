package device_health

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"

	meraki_api "github.com/meraki/dashboard-api-go/v3/sdk"
)

func reportOrganizationDeviceSwitchPortBySwitch(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device, orgSwitchPortsBySwitch *meraki_api.ResponseSwitchGetOrganizationSwitchPortsBySwitch) {

	metrics := []mapstr.M{}

	for _, switchPort := range *orgSwitchPortsBySwitch {

		metric := mapstr.M{
			"switch.port.name":         switchPort.Name,
			"switch.port.serial":       switchPort.Serial,
			"switch.port.mac":          switchPort.Mac,
			"switch.port.network.name": switchPort.Network.Name,
			"switch.port.network.id":   switchPort.Network.ID,
			"switch.port.model":        switchPort.Model,
		}

		for i, port := range *switchPort.Ports {
			metric[fmt.Sprintf("switch.port.item_%d.port_id", i)] = port.PortID
			metric[fmt.Sprintf("switch.port.item_%d.name", i)] = port.Name
			metric[fmt.Sprintf("switch.port.item_%d.tags", i)] = port.Tags
			metric[fmt.Sprintf("switch.port.item_%d.enabled", i)] = port.Enabled
			metric[fmt.Sprintf("switch.port.item_%d.poe_enabled", i)] = port.PoeEnabled
			metric[fmt.Sprintf("switch.port.item_%d.vlan", i)] = port.VLAN
			metric[fmt.Sprintf("switch.port.item_%d.voice_vlan", i)] = port.VoiceVLAN
			metric[fmt.Sprintf("switch.port.item_%d.allowed_vlans", i)] = port.AllowedVLANs
			metric[fmt.Sprintf("switch.port.item_%d.rstp_enabled", i)] = port.RstpEnabled
			metric[fmt.Sprintf("switch.port.item_%d.stp_guard", i)] = port.StpGuard
			metric[fmt.Sprintf("switch.port.item_%d.link_negotiation", i)] = port.LinkNegotiation
			metric[fmt.Sprintf("switch.port.item_%d.access_policy_type", i)] = port.AccessPolicyType
			metric[fmt.Sprintf("switch.port.item_%d.sticky_mac_allow_list", i)] = port.StickyMacAllowList
			metric[fmt.Sprintf("switch.port.item_%d.sticky_mac_allow_list_limit", i)] = port.StickyMacAllowListLimit
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
		metrics = append(metrics, metric)
	}

	ReportMetricsForOrganization(reporter, organizationID, metrics)
}
