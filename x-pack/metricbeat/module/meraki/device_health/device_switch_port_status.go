package device_health

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
	meraki_api "github.com/meraki/dashboard-api-go/v3/sdk"
)

func getSwitchPortStatusBySerial(client *meraki_api.Client, org string) (map[Serial]*meraki_api.ResponseSwitchGetDeviceSwitchPortsStatuses, error) {

	switchSerials_val, switchSerials_res, switchSerials_err := client.Switch.GetOrganizationSwitchPortsBySwitch(org, &meraki_api.GetOrganizationSwitchPortsBySwitchQueryParams{})
	if switchSerials_err != nil {
		return nil, fmt.Errorf("Switch.GetOrganizationSwitchPortsBySwitch failed; [%d] %s. %w", switchSerials_res.StatusCode(), switchSerials_res.Body(), switchSerials_err)
	}

	var switchSerials []string
	for _, switchSerial := range *switchSerials_val {
		switchSerials = append(switchSerials, switchSerial.Serial)
	}

	serialSwitchPortStatuses := make(map[Serial]*meraki_api.ResponseSwitchGetDeviceSwitchPortsStatuses)
	for _, serial := range switchSerials {
		switchPortStatuses_val, switchPortStatuses_res, switchPortStatuses_err := client.Switch.GetDeviceSwitchPortsStatuses(serial, &meraki_api.GetDeviceSwitchPortsStatusesQueryParams{})
		if switchPortStatuses_err != nil {
			return nil, fmt.Errorf("Switch.GetOrganizationSwitchPortsBySwitch failed; [%d] %s. %w", switchPortStatuses_res.StatusCode(), switchPortStatuses_res.Body(), switchPortStatuses_err)
		}
		serialSwitchPortStatuses[Serial(serial)] = switchPortStatuses_val

	}

	return serialSwitchPortStatuses, nil
}

func reportSwitchPortStatusBySerial(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device, switchSerialPortStatuses map[Serial]*meraki_api.ResponseSwitchGetDeviceSwitchPortsStatuses) {
	metrics := []mapstr.M{}

	for serial, portStatuses := range switchSerialPortStatuses {
		for _, portStatus := range *portStatuses {
			metric := mapstr.M{
				"switch.port.status.port_id":                                    portStatus.PortID,
				"switch.port.status.enabled":                                    portStatus.Enabled,
				"switch.port.status.status":                                     portStatus.Status,
				"switch.port.status.spanning_tree.statuses":                     portStatus.SpanningTree.Statuses,
				"switch.port.status.is_uplink":                                  portStatus.IsUplink,
				"switch.port.status.errors":                                     portStatus.Errors,
				"switch.port.status.warnings":                                   portStatus.Warnings,
				"switch.port.status.speed":                                      portStatus.Speed,
				"switch.port.status.duplex":                                     portStatus.Duplex,
				"switch.port.status.usage_in_kb.sent":                           portStatus.UsageInKb.Sent,
				"switch.port.status.usage_in_kb.recv":                           portStatus.UsageInKb.Recv,
				"switch.port.status.usage_in_kb.total":                          portStatus.UsageInKb.Total,
				"switch.port.status.client_count":                               portStatus.ClientCount,
				"switch.port.status.power_usage_in_wh":                          portStatus.PowerUsageInWh,
				"switch.port.status.traffic_in_kbps.total":                      portStatus.TrafficInKbps.Total,
				"switch.port.status.traffic_in_kbps.sent":                       portStatus.TrafficInKbps.Sent,
				"switch.port.status.traffic_in_kbps.recv":                       portStatus.TrafficInKbps.Recv,
				"switch.port.status.secure_port.enabled":                        portStatus.SecurePort.Enabled,
				"switch.port.status.secure_port.active":                         portStatus.SecurePort.Active,
				"switch.port.status.secure_port.authentication_status":          portStatus.SecurePort.AuthenticationStatus,
				"switch.port.status.secure_port.config_overrides.type":          portStatus.SecurePort.ConfigOverrides.Type,
				"switch.port.status.secure_port.config_overrides.vlan":          portStatus.SecurePort.ConfigOverrides.VLAN,
				"switch.port.status.secure_port.config_overrides.voice_vlan":    portStatus.SecurePort.ConfigOverrides.VoiceVLAN,
				"switch.port.status.secure_port.config_overrides.allowed_vlans": portStatus.SecurePort.ConfigOverrides.AllowedVLANs,
				//				"switch.port.status.44":      portStatus.poe.isAllocated  // Missing on meraki go api
			}

			if portStatus.Cdp != nil {
				metric["switch.port.status.cdp.system_name"] = portStatus.Cdp.SystemName
				metric["switch.port.status.cdp.platform"] = portStatus.Cdp.Platform
				metric["switch.port.status.cdp.device_id"] = portStatus.Cdp.DeviceID
				metric["switch.port.status.cdp.port_id"] = portStatus.Cdp.PortID
				metric["switch.port.status.cdp.native_vlan"] = portStatus.Cdp.NativeVLAN
				metric["switch.port.status.cdp.address"] = portStatus.Cdp.Address
				metric["switch.port.status.cdp.management_address"] = portStatus.Cdp.ManagementAddress
				metric["switch.port.status.cdp.version"] = portStatus.Cdp.Version
				metric["switch.port.status.cdp.vtp_management_domain"] = portStatus.Cdp.VtpManagementDomain
				metric["switch.port.status.cdp.capabilities"] = portStatus.Cdp.Capabilities
			}

			if portStatus.Lldp != nil {
				metric["switch.port.status.Lldp.system_name"] = portStatus.Lldp.SystemName
				metric["switch.port.status.Lldp.system_description"] = portStatus.Lldp.SystemDescription
				metric["switch.port.status.Lldp.chassis_id"] = portStatus.Lldp.ChassisID
				metric["switch.port.status.Lldp.port_id"] = portStatus.Lldp.PortID
				metric["switch.port.status.Lldp.management_vlan"] = portStatus.Lldp.ManagementVLAN
				metric["switch.port.status.Lldp.port_vlan"] = portStatus.Lldp.PortVLAN
				metric["switch.port.status.Lldp.management_address"] = portStatus.Lldp.ManagementAddress
				metric["switch.port.status.Lldp.port_description"] = portStatus.Lldp.PortDescription
				metric["switch.port.status.Lldp.system_capabilties"] = portStatus.Lldp.SystemCapabilities
			}

			if device, ok := devices[Serial(serial)]; ok {
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
	}

	ReportMetricsForOrganization(reporter, organizationID, metrics)

}
